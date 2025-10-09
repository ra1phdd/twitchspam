package admin

type fakeAliases struct{}

func (f *fakeAliases) Update(newAliases map[string]string)                    {}
func (f *fakeAliases) ReplaceOne(text string) string                          { return text }
func (f *fakeAliases) ReplacePlaceholders(text string, parts []string) string { return text }

type fakeFileServer struct{}

func (f *fakeFileServer) UploadToHaste(text string) (string, error) { return "key", nil }
func (f *fakeFileServer) GetURL(key string) string                  { return "http://fake/" + key }
