package ports

type FileServerPort interface {
	UploadToHaste(text string) (string, error)
	GetURL(key string) string
}
