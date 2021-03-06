package handler

// // codeMaster前端
// func CodeMasterFontEndHandler(w http.ResponseWriter, r *http.Request) {
// 	url := strings.TrimPrefix(r.URL.Path, "/codeMaster")
// 	logs.Debug("codeMaster font_end: url=%s", url)
// 	if url == "" {
// 		url = "index.html"
// 	}
// 	targetPath := config.ServerConfig.CodeMasterPath + url
// 	http.ServeFile(w, r, targetPath)
// 	return
// }
