package secrets

//go:generate safekeeper --output=appsecrets.go --keys=DROPBOX_CLIENT_ID,DROPBOX_CLIENT_SECRET $GOFILE
