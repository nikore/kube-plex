package kube

import (
	"fmt"
	"github.com/nikore/kube-plex/pkg/signals"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"log"
	"strings"
	"time"
)

type Transcoder interface {
	Namespace(namespace string) Transcoder
	Image(image string) Transcoder
	WorkingDir(workingDir string) Transcoder
	Command(command []string) Transcoder
	EnvVars(env []string) Transcoder
	DataPVC(dataPVC string) Transcoder
	BuildPod() *corev1.Pod
	RunAndWait() error
}

type transcoder struct {
	client Client
	namespace string
	image string
	workingDir string
	command []string
	envVars []corev1.EnvVar
	dataPVC string
}

func NewTranscoder(client Client) Transcoder {
	return &transcoder{
		client: client,
		namespace: "default",
		image: "plexinc/pms-docker:latest",
		workingDir: ".",
		dataPVC: "plex-pvc",
	}
}

func (t *transcoder) Namespace(namespace string) Transcoder {
	t.namespace = namespace
	return t
}

func (t *transcoder) Image(image string) Transcoder {
	t.image = image
	return t
}

func (t *transcoder) WorkingDir(workingDir string) Transcoder {
	t.workingDir = workingDir
	return t
}

func (t *transcoder) Command(command []string) Transcoder {
	t.command = command
	return t
}

func (t *transcoder) EnvVars(env []string) Transcoder {
	out := make([]corev1.EnvVar, len(env))
	for _, v := range env {
		splitvar := strings.SplitN(v, "=", 2)
		out = append(out, corev1.EnvVar{Name: splitvar[0], Value: splitvar[1]})
	}

	t.envVars = out
	return t
}

func (t *transcoder) DataPVC(dataPVC string) Transcoder {
	t.dataPVC = dataPVC
	return t
}

func (t *transcoder) BuildPod() *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "pms-elastic-transcoder-",
		},
		Spec: corev1.PodSpec{
			NodeSelector: map[string]string{
				"beta.kubernetes.io/arch": "amd64",
			},
			RestartPolicy: corev1.RestartPolicyNever,
			Containers: []corev1.Container{
				{
					Name:       "plex",
					Command:    t.command,
					Image:      t.image,
					Env:        t.envVars,
					WorkingDir: t.workingDir,
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "data",
							MountPath: "/data",
							ReadOnly:  true,
						},
						{
							Name:      "config",
							MountPath: "/config",
						},
						{
							Name:      "transcode",
							MountPath: "/transcode",
						},
					},
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: "data",
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: t.dataPVC,
						},
					},
				},
				{
					Name: "config",
					VolumeSource: corev1.VolumeSource {
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
				{
					Name: "transcode",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			},
		},
	}
}

func (t *transcoder) RunAndWait() error {
	podClient := t.client.Get().CoreV1()

	pod, err := podClient.Pods(t.namespace).Create(t.BuildPod())
	if err != nil {
		return err
	}

	stopCh := signals.SetupSignalHandler()
	waitFn := func() <-chan error {
		stopCh := make(chan error)
		go func() {
			stopCh <- t.waitForPodCompletion(t.client.Get(), pod)
		}()
		return stopCh
	}

	select {
	case err := <-waitFn():
		if err != nil {
			return err
		}
	case <-stopCh:
		return err
	}

	log.Printf("Cleaning up pod...")
	err = podClient.Pods(t.namespace).Delete(pod.Name, nil)
	if err != nil {
		return err
	}

	return nil
}

func (t *transcoder) waitForPodCompletion(cl kubernetes.Interface, pod *corev1.Pod) error {
	for {
		pod, err := cl.CoreV1().Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		switch pod.Status.Phase {
		case corev1.PodPending:
		case corev1.PodRunning:
		case corev1.PodUnknown:
			log.Printf("Warning: pod %q is in an unknown state", pod.Name)
		case corev1.PodFailed:
			return fmt.Errorf("pod %q failed", pod.Name)
		case corev1.PodSucceeded:
			return nil
		}
		time.Sleep(1 * time.Second)
	}
}
