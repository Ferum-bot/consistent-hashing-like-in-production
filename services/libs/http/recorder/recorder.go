package recorder

import "net/http"

type Recorder struct {
	http.ResponseWriter
	status int
}

func New(w http.ResponseWriter) *Recorder {
	return &Recorder{ResponseWriter: w, status: http.StatusOK}
}

func (r *Recorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *Recorder) Status() int {
	return r.status
}

func (r *Recorder) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}
