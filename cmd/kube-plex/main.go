package main

import (
	"github.com/nikore/kube-plex/pkg/kube"
	"log"
	"os"
	"strings"
)

// data pvc name
var dataPVC = os.Getenv("DATA_PVC")

// pms namespace
var namespace = os.Getenv("KUBE_NAMESPACE")

// image for the plexmediaserver container containing the transcoder. This
// should be set to the same as the 'master' pms server
var pmsImage = os.Getenv("PMS_IMAGE")
var pmsInternalAddress = os.Getenv("PMS_INTERNAL_ADDRESS")

func main() {
	env := os.Environ()
	args := os.Args

	rewriteEnv(env)
	rewriteArgs(args)
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Error getting working directory: %s", err)
	}
	transcoderPod := kube.NewTranscoder(kube.NewClient()).Namespace(namespace).Image(pmsImage).WorkingDir(cwd).EnvVars(env).Command(args).DataPVC(dataPVC)

	err = transcoderPod.RunAndWait()
	if err != nil {
		log.Fatalf("Error creating pod: %s", err)
	}

}

// rewriteEnv rewrites environment variables to be passed to the transcoder
func rewriteEnv(in []string) {
	// no changes needed
}

func rewriteArgs(in []string) {
	for i, v := range in {
		switch v {
		case "-progressurl", "-manifest_name", "-segment_list":
			in[i+1] = strings.Replace(in[i+1], "http://127.0.0.1:32400", pmsInternalAddress, 1)
		case "-loglevel", "-loglevel_plex":
			in[i+1] = "debug"
		}
	}
}