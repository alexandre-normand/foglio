package main

//go:generate giddyup

import (
	"bufio"
	"fmt"
	"github.com/alexandre-normand/foglio/secrets"
	"github.com/alexkappa/mustache"
	"github.com/dropbox/dropbox-sdk-go-unofficial/dropbox"
	"github.com/dropbox/dropbox-sdk-go-unofficial/dropbox/files"
	"github.com/dropbox/dropbox-sdk-go-unofficial/dropbox/sharing"
	"github.com/mitchellh/go-homedir"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"gopkg.in/alecthomas/kingpin.v2"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

const (
	MAIN_TEMPLATE     = "main"
	SMALL_SIZE_SUFFIX = "-small"
	TOKEN_FILE_PATH   = "~/.foglioToken"
)

var (
	templateFile    = kingpin.Flag("template", "Path to template file to generate the output.").Required().String()
	outputDirectory = kingpin.Flag("outputDirectory", "Path to output directory for the generated files.").Required().String()
)

type PortfolioElement struct {
	SmallSizeLink string
	LargeSizeLink string
	Title         string
}

func main() {
	kingpin.Version(VERSION)
	kingpin.Parse()

	templateContent, err := ioutil.ReadFile(*templateFile)
	if err != nil {
		fmt.Printf("Error parsing template file [%s]: %v\n", *templateFile, err)
		os.Exit(-1)
	}

	template := mustache.New()
	template.Option(mustache.Delimiters("[[", "]]"))
	err = template.ParseString(string(templateContent))
	if err != nil {
		fmt.Printf("Error generating template from [%s]: %v\n", templateContent, err)
		os.Exit(-1)
	}

	outputDir, err := homedir.Expand(*outputDirectory)
	if err != nil {
		fmt.Printf("Error expanding output directory [%s]: %v\n", *outputDirectory, err)
		os.Exit(-1)
	}

	token, err := getAccessToken()
	if err != nil {
		fmt.Printf("Error getting access token: %v\n", err)
		os.Exit(-1)
	}

	config := dropbox.Config{Token: token, Verbose: false}

	directory := "/photo.heyitsalex.net"

	files, err := getAllFilesInDirectory(config, directory)
	if err != nil {
		fmt.Printf("Error getting files in directory [%s]: %v\n", directory, err)
		os.Exit(-1)
	}

	accessLinks, err := getAccessLinks(config, files)
	if err != nil {
		fmt.Printf("Error creating access links: %v\n", err)
		os.Exit(-1)
	}

	portfolioElements := getPortfolioElements(accessLinks)

	err = generatePosts(template, portfolioElements, outputDir)
	if err != nil {
		fmt.Printf("Error generating posts: %v", err)
		os.Exit(-1)
	}
}

func getAllFilesInDirectory(config dropbox.Config, directory string) ([]*files.FileMetadata, error) {
	dbx := files.New(config)

	r, err := dbx.ListFolder(files.NewListFolderArg(directory))
	if err != nil {
		return nil, fmt.Errorf("Error listing directory [%s]: %v", directory, err)
	}

	entries := r.Entries
	entries = append(entries, r.Entries...)
	for r.HasMore {
		arg := files.NewListFolderContinueArg(r.Cursor)

		r, err = dbx.ListFolderContinue(arg)
		if err != nil {
			return nil, err
		}

		entries = append(entries, r.Entries...)
	}

	var listing []*files.FileMetadata
	for _, entry := range entries {
		if f, ok := entry.(*files.FileMetadata); ok {
			listing = append(listing, f)
		}
	}

	return listing, nil
}

func getAccessLinks(config dropbox.Config, files []*files.FileMetadata) ([]*sharing.FileLinkMetadata, error) {
	dbx := sharing.New(config)
	var sharedFiles []*sharing.FileLinkMetadata

	for _, file := range files {

		listLinks := sharing.NewListSharedLinksArg()
		listLinks.Path = file.PathLower

		lr, err := dbx.ListSharedLinks(listLinks)
		if err != nil {
			return nil, fmt.Errorf("Error getting shared link for [%s]: %v\n%s", file.Name, err)
		} else {
			if len(lr.Links) == 0 {
				fmt.Printf("Creating shared link for file: %v\n", file.Name)

				r, err := dbx.CreateSharedLinkWithSettings(sharing.NewCreateSharedLinkWithSettingsArg(file.PathLower))
				if err != nil {
					return nil, fmt.Errorf("Error creating shared link for [%s]: %v\n%s", file.Name, err)
				}

				if sf, ok := r.(*sharing.FileLinkMetadata); ok {
					sharedFiles = append(sharedFiles, sf)
				}
			} else {
				for _, sl := range lr.Links {
					if fsl, ok := sl.(*sharing.FileLinkMetadata); ok {
						sharedFiles = append(sharedFiles, fsl)
					}
				}
				for lr.HasMore {
					arg := sharing.NewListSharedLinksArg()
					arg.Cursor = lr.Cursor
					arg.Path = file.PathLower

					lr, err := dbx.ListSharedLinks(arg)
					if err != nil {
						return nil, err
					}

					for _, sl := range lr.Links {
						if fsl, ok := sl.(*sharing.FileLinkMetadata); ok {
							sharedFiles = append(sharedFiles, fsl)
						}
					}
				}
			}
		}
	}

	return sharedFiles, nil
}

func getPortfolioElements(accessLinks []*sharing.FileLinkMetadata) []PortfolioElement {
	var portfolioElements map[string]PortfolioElement
	portfolioElements = make(map[string]PortfolioElement)

	for _, al := range accessLinks {
		name := strings.TrimSuffix(al.Name, ".jpg")
		name = strings.TrimSuffix(name, ".png")
		logicalName := strings.TrimSuffix(name, SMALL_SIZE_SUFFIX)

		titleName := strings.Title(logicalName)

		normalizedViewableLink := strings.Replace(al.Url, "www.dropbox.com", "dl.dropboxusercontent.com", 1)
		normalizedViewableLink = strings.TrimSuffix(normalizedViewableLink, "?dl=0")

		if element, ok := portfolioElements[titleName]; ok {
			if strings.HasSuffix(name, "-small") {
				element.SmallSizeLink = normalizedViewableLink
			} else {
				element.LargeSizeLink = normalizedViewableLink
			}

			portfolioElements[titleName] = element
		} else {
			newElement := PortfolioElement{Title: titleName, SmallSizeLink: "", LargeSizeLink: ""}
			if strings.HasSuffix(name, "-small") {
				element.SmallSizeLink = normalizedViewableLink
			} else {
				element.LargeSizeLink = normalizedViewableLink
			}

			portfolioElements[titleName] = newElement
		}
	}

	var elements []PortfolioElement
	for _, v := range portfolioElements {
		elements = append(elements, v)
	}

	return elements
}

func generatePosts(template *mustache.Template, elements []PortfolioElement, outputDir string) error {
	for _, e := range elements {
		normalizedName := strings.Replace(strings.ToLower(e.Title), " ", "-", -1) + ".md"
		description := strings.ToLower(e.Title)

		if e.SmallSizeLink == "" || e.LargeSizeLink == "" {
			fmt.Fprintf(os.Stderr, "Skipping [%s] because of missing links: small: [%s], large: [%s]\n", e.Title, e.SmallSizeLink, e.LargeSizeLink)
		} else {
			data, err := template.RenderString(map[string]string{"name": e.Title, "smallSizeLink": e.SmallSizeLink, "largeSizeLink": e.LargeSizeLink, "description": description})
			if err != nil {
				return fmt.Errorf("Error rendering template for [%s]: %v\n", normalizedName, err)
			}

			outputPath := filepath.Join(outputDir, normalizedName)
			err = ioutil.WriteFile(outputPath, []byte(data), 0644)
			if err != nil {
				return fmt.Errorf("Error writing rendered templated to file [%s]: %v\n", outputPath, err)
			}

			fmt.Printf("Rendered template for [%s] to [%s]\n", normalizedName, outputPath)
		}
	}

	return nil
}

func getAccessToken() (string, error) {
	tokenFile, err := homedir.Expand(TOKEN_FILE_PATH)
	if err != nil {
		return "", fmt.Errorf("Error expanding path for [%s] file: %v", TOKEN_FILE_PATH, err)
	}

	tokenFileContent, err := ioutil.ReadFile(tokenFile)
	if err == nil && len(tokenFileContent) > 0 {
		return string(tokenFileContent), nil
	}

	secrets := secrets.NewAppSecrets()

	ctx := context.Background()
	conf := &oauth2.Config{
		ClientID:     secrets.DropboxClientId,
		ClientSecret: secrets.DropboxClientSecret,
		Scopes:       []string{},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://www.dropbox.com/oauth2/authorize",
			TokenURL: "https://api.dropboxapi.com/1/oauth2/token",
		},
	}

	// Redirect user to consent page to ask for permission
	// for the scopes specified above.
	url := conf.AuthCodeURL("state")
	fmt.Printf("Visit the URL for the auth dialog: %v\n", url)

	// Use the authorization code that is pushed to the redirect
	// URL. Exchange will do the handshake to retrieve the
	// initial access token. The HTTP Client returned by
	// conf.Client will refresh the token as necessary.
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter code: ")
	code, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("Error getting code: %v", err)
	}

	code = strings.TrimSpace(code)

	if tok, err := conf.Exchange(ctx, code); err != nil {
		return "", fmt.Errorf("Error getting token from code [%s]: %v", code, err)
	} else {
		// Save token
		err := ioutil.WriteFile(tokenFile, []byte(tok.AccessToken), 0644)

		if err != nil {
			fmt.Errorf("Error saving token to file [%s]: %v", tokenFile, err)
		}

		return tok.AccessToken, nil
	}
}
