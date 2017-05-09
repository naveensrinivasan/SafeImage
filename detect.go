package main

import (
	b64 "encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	vision "cloud.google.com/go/vision"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rekognition"
	"golang.org/x/net/context"
	"google.golang.org/api/option"
)

var (
	sess *session.Session
	svc  *rekognition.Rekognition
)

type hashSet struct {
	data map[string]bool
}

func (hash *hashSet) Add(value string) {
	hash.data[strings.ToLower(value)] = true
}

func (hash *hashSet) Contains(value string) (exists bool) {
	_, exists = hash.data[strings.ToLower(value)]
	return
}

func (hash *hashSet) Length() int {
	return len(hash.data)
}
func (hash *hashSet) RemoveDuplicates() {}

func newSet() *hashSet {
	return &hashSet{make(map[string]bool)}
}
func init() {
	var err error
	sess, err = session.NewSession(&aws.Config{Region: aws.String("us-east-1")})
	if err != nil {
		fmt.Println("failed to create session,", err)
		return
	}
	svc = rekognition.New(sess)
}
func main() {
	imagesdir := "/Users/naveen/Downloads/safeimage/"
	set := newSet()
	getBadWords(set)

	files, err := ioutil.ReadDir(imagesdir)

	if err != nil {
		log.Fatal(err)
	}

	for _, file := range files {
		fullName := filepath.Join(imagesdir, file.Name())
		annotation, err := detectSafeSearch(os.Stdout, fullName)
		texts, errors := detectText(fullName)

		if err != nil {
			log.Fatal(err)
		}
		if errors != nil {
			log.Fatal(errors)
		}

		for _, text := range texts {
			if set.Contains(text.Description) {
				fmt.Fprintf(os.Stdout, "File : %s has inappropriate language %s \n", fullName, text.Description)
			}
		}
		awsimagevalidation(fullName)
		dumpResults(os.Stdout, annotation, file.Name())
	}
}

func awsimagevalidation(filename string) error {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	encoded := make([]byte, b64.StdEncoding.EncodedLen(len(b)))
	b64.StdEncoding.Encode(encoded, b)
	request := &rekognition.DetectModerationLabelsInput{
		Image: &rekognition.Image{ // Required
			Bytes: b,
		},

		//	MinConfidence: aws.Float64(10)
	}
	result, err := svc.DetectModerationLabels(request)
	if err != nil {
		log.Fatal(err)
	}
	for _, item := range result.ModerationLabels {
		fmt.Println("aws :- ", item.Name)
	}
	return nil

}

func getBadWords(hashset *hashSet) {
	badwordfiles := []string{"en.txt", "fr.txt"}
	for _, f := range badwordfiles {
		badwordsdir := "/Users/naveen/Downloads/badwords/"
		bytes, err := ioutil.ReadFile(filepath.Join(badwordsdir, f))

		if err != nil {
			log.Fatal(err)
		}
		for _, word := range strings.Split(string(bytes), "\n") {
			hashset.Add(word)
		}
	}
}
func dumpResults(w io.Writer, annotation *vision.SafeSearchAnnotation, filename string) {

	if annotation.Adult > 2 || annotation.Medical > 2 || annotation.Spoof > 2 || annotation.Violence > 2 {
		fmt.Fprint(w, "File :", filename)
	}
	if annotation.Adult == 3 {
		fmt.Fprintln(w, " Is likely adult content")

	}
	if annotation.Adult > 3 {
		fmt.Fprintln(w, " Is adult content")
	}

	if annotation.Medical == 3 {
		fmt.Fprintln(w, " Is likely medical content")
	}
	if annotation.Adult > 3 {
		fmt.Fprintln(w, " Is Medical content")
	}

	if annotation.Violence == 3 {
		fmt.Fprintln(w, " Is likely Violence content")
	}
	if annotation.Adult > 3 {
		fmt.Fprintln(w, " Is Violence content")
	}

	if annotation.Spoof == 3 {
		fmt.Fprintln(w, " Is likely Spoof content")
	}
	if annotation.Spoof > 3 {
		fmt.Fprintln(w, " Is Spoof content")
	}
}

// detectText gets text from the Vision API for an image at the given file path.
func detectText(file string) ([]*vision.EntityAnnotation, error) {
	ctx := context.Background()

	client, err := vision.NewClient(ctx, option.WithServiceAccountFile("/Users/naveen/Downloads/playground-4ae5b3036c38.json"))
	if err != nil {
		return nil, err
	}

	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	image, err := vision.NewImageFromReader(f)
	if err != nil {
		return nil, err
	}
	return client.DetectTexts(ctx, image, 20)

}

// detectSafeSearch gets image properties from the Vision API for an image at the given file path.
func detectSafeSearch(w io.Writer, file string) (*vision.SafeSearchAnnotation, error) {
	ctx := context.Background()

	client, err := vision.NewClient(ctx, option.WithServiceAccountFile("/Users/naveen/Downloads/playground-4ae5b3036c38.json"))
	if err != nil {
		return nil, err
	}

	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	image, err := vision.NewImageFromReader(f)
	if err != nil {
		return nil, err
	}
	return client.DetectSafeSearch(ctx, image)
}
