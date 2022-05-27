package callback

import (
	"io/ioutil"
	"net/http"
	"strings"
	"time"
	"vsphere-facade/api/e"
	"vsphere-facade/app/logging"
	"vsphere-facade/app/utils"
	"vsphere-facade/config"
	"vsphere-facade/vsphere/protocol"
)

type Callbacker struct {
	Req protocol.CallbackReq
}

func NewCallbacker(req protocol.CallbackReq) *Callbacker {
	return &Callbacker{
		Req: req,
	}
}

func (c *Callbacker) CallbackObj(requestID string, data interface{}) {
	if data == nil {
		data = e.EmptyObject()
	}
	c.callback(requestID, data, nil)
}

func (c *Callbacker) CallbackArr(requestID string, data interface{}) {
	if data == nil {
		data = e.EmptyArray()
	}
	c.callback(requestID, data, nil)
}

func (c *Callbacker) CallbackErr(requestID string, data interface{}, err error) {
	c.callback(requestID, data, err)
}

func (c *Callbacker) callback(requestID string, data interface{}, err error) {
	res := protocol.CallbackRes{
		RequestID: requestID,
		Data:      data,
	}

	if err != nil {
		res.Code = e.FAILED
		res.Message = err.Error()
	} else {
		res.Code = e.Success
	}
	_ = c.sendByHttp(res)
}

func (c *Callbacker) sendByHttp(res protocol.CallbackRes) error {
	cb := utils.NilNext(c.Req.HttpPost, config.G.Vsphere.Default.Callback.HttpPost)
	httpCB := cb.(*protocol.Http)
	if httpCB == nil {
		return nil
	}

	body := utils.ToJson(res)
	logging.L().Debugf("http回调：\nPOST %s \nHeaders: %s \nPayload: %s", httpCB.URL, httpCB.Headers, body)
	post, err := http.NewRequest("POST", httpCB.URL, strings.NewReader(body))
	if err != nil {
		logging.L().Error("http post回调时，创建http客户端失败", err)
		return err
	}

	// Header
	post.Header.Add("Content-Type", "application/json")
	for k, v := range httpCB.Headers {
		post.Header.Add(k, v)
	}
	// Send
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	resp, err := client.Do(post)
	if err != nil {
		logging.L().Errorf("http post回调时，请求回调URL失败\n: POST %s\n %s\n %v", httpCB.URL, body, err)
		return err
	}

	if logging.IsDebug() {
		defer resp.Body.Close()
		rbody, _ := ioutil.ReadAll(resp.Body)
		logging.L().Debugf("响应状态: %d\n响应内容: %s", resp.StatusCode, string(rbody))
	}

	if resp.StatusCode > 399 {
		defer resp.Body.Close()
		rbody, _ := ioutil.ReadAll(resp.Body)
		logging.L().Errorf("http post回调后，后响应错误\n: POST %s\n %s\n响应状态: %d\n响应内容: %s",
			httpCB.URL, body, resp.StatusCode, string(rbody))
	}
	return nil
}
