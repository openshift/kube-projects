package bootstraproutes

import (
	"bytes"
	"net/http"
	"text/template"
)

type APIFederation struct {
	InternalHost string
	CABundle     []byte
}

// Install adds the Index webservice to the given mux.
func (i APIFederation) Install(mux *http.ServeMux) {
	mux.HandleFunc("/bootstrap/apifederation", func(w http.ResponseWriter, r *http.Request) {
		apifederationTemplate, err := template.New("apifederationTemplate").Parse(apifederationJSON)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		buffer := &bytes.Buffer{}
		if err := apifederationTemplate.Execute(buffer, i); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Write(buffer.Bytes())
	})
}

const apifederationJSON = `{
	"apiVersion": "apifederation.openshift.io/v1beta1",
	"kind": "APIServer",
	"metadata": {
		"name": "v1.project.openshift.io"
	},
	"spec": {
		"group": "project.openshift.io",
		"version": "v1",
		"internalHost": "{{.InternalHost}}",
		"insecureSkipTLSVerify": true,
		"priority": 2
	}
}`
