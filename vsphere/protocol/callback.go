package protocol

type CallbackReq struct {
	HttpPost *Http `json:"httpPost,omitempty" mapstructure:"httpPost"`

	RequestID string `json:"-"`
}

type CallbackRes struct {
	RequestID string      `json:"requestId"`
	Code      string      `json:"code"`
	Message   string      `json:"message,omitempty"`
	Data      interface{} `json:"data,omitempty"`
}

type Http struct {
	URL     string            `json:"url,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}
