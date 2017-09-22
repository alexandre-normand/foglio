# foglio
Very custom-tailored command line tool that generates markdown files with proper metadata from a dropbox directory with portfolio pictures

Why
===
I've had to change my portfolio image links a few times already and since I'm statically generating markdown for each, it's a bit annoying to have to do that manually. And there's also the part that I don't trust myself to not make mistakes and forget some when I'm doing that myself. 

Having a tool that generates the markdown also means that I can potentially change the tool to read images from a new source if I ever have to change where I'm hosting my images. 

Integration example
===================

My portfolio uses this to generate a post for each image in a dropbox directory. Here's what a [template file](https://raw.githubusercontent.com/alexandre-normand/photofolio/master/template.md) looks like:

```
+++
showonlyimage = true
draft = false
image = "[[smallSizeLink]]"
title = "[[name]]"
weight = 1
+++

{{% fig class="full" src="[[ largeSizeLink ]]" for-sale="true" %}}

```

Running the tool is done like this: 

```
./foglio --template <path-to-template>/template.md --outputDirectory <path-to-destination-content>/content/portfolio/
```

Pre-requisites for compilation
==============================

This repository doesn't include the client id and secret to access dropbox. In order to get the proper credentials added to compile this, you'll need to run `go generate` once to generate the `appsecrets.go` that will include those. Here's an example:

```
DROPBOX_CLIENT_ID=<YOUR_CLIENT_ID> DROPBOX_CLIENT_SECRET=<YOUR_CLIENT_SECRET> go generate ./secrets/secrets.go
```

Note that you only need to do this when you get new values or if you delete `appsecrets.go`. Note that the generated code with the secrets is in the `.gitignore` and should remain that way to avoid secrets leaking into `git`.

