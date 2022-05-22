package blog

import "net/http"

func (env *Env) PostsGet(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("PostsGet"))
}
