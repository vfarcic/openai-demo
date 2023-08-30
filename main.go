package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	openai "github.com/sashabaranov/go-openai"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var rootCmd = &cobra.Command{
	Use:   "k8sai",
	Short: "k8sai is a super fancy CLI for Kubernetes AI.",
	Run:   func(cmd *cobra.Command, args []string) {},
}

type Settings struct {
	Namespace string
}

var settings Settings

func init() {
	rootCmd.Flags().StringVar(&settings.Namespace, "namespace", "", "Namespace, please (required)")
	rootCmd.MarkFlagRequired("namespace")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Whoops. There was an error while executing the CLI '%s'", err)
		os.Exit(1)
	}
	pods := getPendingPods()
	for _, pod := range pods {
		q := "Explain what's wrong with a Kubernetes pod that contains following events:"
		events := getPodEvents(pod.Name)
		for _, event := range events {
			q = fmt.Sprintf("%s\n%s", q, event)
		}
		aiAnswer := askAI(q)
		answer := fmt.Sprintf("\nHere's the solution for the issues with pod %s:\n\n\n%s\n\n-------------------------------\n\n", pod.Name, aiAnswer)
		println(answer)
	}
}

func getPendingPods() []v1.Pod {
	pods := []v1.Pod{}
	for _, pod := range getPods() {
		if pod.Status.Phase == v1.PodPending {
			pods = append(pods, pod)
		}
	}
	return pods
}

func getPods() []v1.Pod {
	c := getClientset()
	pods, err := c.CoreV1().Pods(settings.Namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	return pods.Items
}

func getPodEvents(podName string) []string {
	events := []string{}
	c := getClientset()
	eventList, err := c.CoreV1().Events(settings.Namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	for _, event := range eventList.Items {
		if event.InvolvedObject.Kind == "Pod" && event.InvolvedObject.Name == podName && event.InvolvedObject.Namespace == settings.Namespace {
			events = append(events, event.Message)
		}
	}
	return events
}

func getClientset() *kubernetes.Clientset {
	kubeConfig := fmt.Sprintf("%s/.kube/config", os.Getenv("HOME"))
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfig)
	if err != nil {
		panic(err.Error())
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	return clientset
}

func askAI(question string) string {
	key := os.Getenv("OPENAI_KEY")
	client := openai.NewClient(key)
	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: question,
				},
			},
		},
	)
	if err != nil {
		fmt.Printf("ChatCompletion error: %v\n", err)
		return ""
	}
	respString := strings.ReplaceAll(resp.Choices[0].Message.Content, "\"", "")
	return respString
}
