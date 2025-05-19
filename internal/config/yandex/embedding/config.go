package embedding

import "os"

const (
	MODEL_URI string = "emb://b1g7e364b5giim9tajta/text-search-doc/latest"
	FOLDER_ID string = "b1g7e364b5giim9tajta"
	URL       string = "https://llm.api.cloud.yandex.net:443/foundationModels/v1/textEmbedding"
)

var TOKEN = os.Getenv("YANDEX_TOKEN")

type Request struct {
	ModelURI string `json:"modelUri"`
	Text     string `json:"text"`
}

type Response struct {
	Embedding    []float64 `json:"embedding"`
	NumTokens    string    `json:"numTokens"`
	ModelVersion string    `json:"modelVersion"`
}
