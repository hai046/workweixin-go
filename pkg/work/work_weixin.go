package work

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
	"work-wechat/pkg/work/wxbizmsgcrypt"
)

type EnvConfig struct {
	Port       string      `yaml:"port"`
	ChatConfig *ChatConfig `yaml:"chat"`
}

type ChatConfig struct {
	Corpid                 string `yaml:"corpid"`
	Secret                 string `yaml:"secret"`
	AgentId                int    `yaml:"agentId"`
	CallBackToken          string `yaml:"callbackToken"`
	CallBackEncodingAESKey string `yaml:"callbackEncodingAESKey"`
}

type WorkWeixin struct {
	corpid        string
	corpsecret    string
	token         *AccessToken
	agentId       int
	chatConfig    *ChatConfig
	userMobiles   map[string]string
	existGroupIds []string //存在的群组

}

type AccessToken struct {
	Access_token string `json:"access_token"`
	Expires_in   int64  `json:"expires_in"` //秒 默认返回是2小时
	ExpireAt     int64  `json:"expireAt"`   //在什么时候过期
	Errcode      int    `json:"errcode"`
	Errmsg       string `json:"errmsg"`
}

type Department struct {
	Id       int    `json:"id"`
	Name     string `json:"name"`
	Parentid int32  `json:"parentid"`
	Order    int32  `json:"order"`
}

type ResponseBase struct {
	Errcode int    `json:"errcode"`
	Errmsg  string `json:"errmsg"`
}

type departments struct {
	ResponseBase
	Department []Department `json:"department"`
}

type users struct {
	ResponseBase
	Userlist []User `json:"userlist"`
}

type User struct {
	Userid       string `json:"userid"`
	Name         string `json:"name"`
	Department   []int  `json:"department"`
	Mobile       string `json:"mobile"`
	Email        string `json:"email"`
	Status       int    `json:"status"`
	Avatar       string `json:"avatar"`
	Telephone    string `json:"telephone"`
	English_name string `json:"english_name"`
}

type requestChatInfo struct {
	//docs https://work.weixin.qq.com/api/doc/90000/90135/90245
	Name     string   `json:"name"`
	UserList []string `json:"userlist"`
	ChatId   string   `json:"chatid"`
	Owner    string   `json:"owner"`
}

type requestChatContent struct {
	Content string `json:"content"`
}
type requestChatMarkdownMsg struct {
	//docs https://work.weixin.qq.com/api/doc/90000/90135/90245
	Msgtype string              `json:"msgtype"`
	ChatId  string              `json:"chatid"`
	Content *requestChatContent `json:"markdown"`
}
type responseChatInfo struct {
	ResponseBase
	ChatId string `json:"chatid"`
}

//docs https://work.weixin.qq.com/api/doc/90000/90135/90236#%E6%96%87%E6%9C%AC%E5%8D%A1%E7%89%87%E6%B6%88%E6%81%AF
type TemplateCard struct {
	MsgType string `json:"msgtype"`
	ToUser  string `json:"touser"`
	//ToTag   string       `json:"totag"`
	AgentId int          `json:"agentid"`
	Content *interface{} `json:"template_card"`
}

//type TemplateCardBody struct {
//	CardType string `json:"card_type"`
//}

//agentId 表示应用id， 0  表示本应用
//微信
func (w *WorkWeixin) Init(conf *ChatConfig) {
	w.corpid = conf.Corpid
	w.corpsecret = conf.Secret
	w.agentId = conf.AgentId
	w.chatConfig = conf
	w.GetAccessToken()

}

type Tag struct {
	TagName string `json:"tagname"`
	TagId   int    `json:"tagid"`
}
type tags struct {
	Errcode int    `json:"errcode"`
	Errmsg  string `json:"errmsg"`
	TagList []Tag  `json:"taglist"`
}

func (w *WorkWeixin) GetDepartmentParentList() ([]Department, error) {

	return w.getDepartmentList(1)
}

func (w *WorkWeixin) GetSonDepartmentParentList(sonDepartment int) ([]Department, error) {

	return w.getDepartmentList(sonDepartment)
}

//获取部门成员
func (w *WorkWeixin) getDepartmentList(sonDepartment int) ([]Department, error) {
	url := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/department/list?access_token=%s&id=%d", w.GetAccessToken(), sonDepartment)
	buffer, err := GetRequestUrl(url)
	if err != nil {
		log.Panic(err)
	}
	var f departments
	err = json.Unmarshal(buffer, &f)
	if err != nil {
		log.Println("getDepartmentList Unmarshal err=", err)
	}

	log.Println(f)

	return f.Department, nil
}

//department_id 部门id
//fetch_child 1/0：是否递归获取子部门下面的成员
func (w *WorkWeixin) GetDepartmentUsers(department_id int, fetch_child int) []User {

	url := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/user/list?access_token=%s&department_id=%d&fetch_child=%d",
		w.GetAccessToken(), department_id, fetch_child)
	buffer, err := GetRequestUrl(url)
	if err != nil {
		log.Panic(err)
	}
	var f users
	err = json.Unmarshal(buffer, &f)
	if err != nil {
		log.Println("GetDepartmentUsers Unmarshal err=", err)
	}

	return f.Userlist
}

//参数	必须		说明
//access_token	是	调用接口凭证
//tagname	是	标签名称，长度限制为32个字（汉字或英文字母），标签名不可与其他标签重名。
//tagid	否	标签id，非负整型，指定此参数时新增的标签会生成对应的标签id，不指定时则以目前最大的id自增。
func (w *WorkWeixin) CreateTag(tag Tag) {

	url := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/tag/create?access_token=%s",
		w.GetAccessToken())

	body, err := json.Marshal(tag)
	if err != nil {
		fmt.Println("Marshal msg err", err)
		return
	}
	buffer, err := PostRequestUrl(url, bytes.NewBuffer(body))
	if err != nil {
		log.Panic(err)
	}
	log.Println("CreateTag Unmarshal err=", err, string(buffer))

}

//获取标签
func (w *WorkWeixin) GetTagList() []Tag {

	url := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/tag/list?access_token=%s",
		w.GetAccessToken())

	buffer, err := GetRequestUrl(url)
	if err != nil {
		log.Panic(err)
	}

	var f tags
	err = json.Unmarshal(buffer, &f)
	if err != nil {
		log.Println("GetDepartmentUsers Unmarshal err=", err)
	}
	return f.TagList
}
func (w *WorkWeixin) GetTagUser(tagid int) []User {

	url := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/tag/get?access_token=%s&tagid=%d",
		w.GetAccessToken(), tagid)

	buffer, err := GetRequestUrl(url)
	if err != nil {
		log.Panic(err)
	}

	var f users
	err = json.Unmarshal(buffer, &f)
	if err != nil {
		log.Println("GetTagUser Unmarshal err=", err)
	}
	return f.Userlist
}

//添加标签成员
func (w *WorkWeixin) AddTagUsers(userIds []string, tagId int) string {

	url := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/tag/addtagusers?access_token=%s",
		w.GetAccessToken())

	bodyStruct := map[string]interface{}{
		"tagid":    tagId,
		"userlist": userIds,
	}

	body, err := json.Marshal(bodyStruct)
	log.Println("json", string(body))
	if err != nil {
		fmt.Println("Marshal msg err", err)
		return fmt.Sprint(err)
	}
	buffer, err := PostRequestUrl(url, bytes.NewBuffer(body))

	if err != nil {
		log.Panic(err)
	}

	return string(buffer)
}

func (w *WorkWeixin) SendText(toUser string, toparties string, totag string, text string) string {

	log.Println("send msg=", text)
	bodyStruct := map[string]interface{}{
		"touser":  toUser,
		"toparty": toparties,
		"totag":   totag,
		"msgtype": "text",
		"agentid": w.agentId,
		"text": map[string]interface{}{
			"content": text,
		},
	}

	body, err := json.Marshal(bodyStruct)

	if err != nil {
		fmt.Println("Marshal msg err", err)
		return "Marshal msg err"
	}
	log.Println(string(body))

	buffer, err := PostRequestUrl(fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/message/send?access_token=%s",
		w.GetAccessToken()),
		bytes.NewBuffer(body))
	//
	if err != nil {
		log.Println("send text err", err)
		return fmt.Sprint(err)
	} else {
		log.Println(string(buffer))
		return string(buffer)
	}

}

func (w *WorkWeixin) checkToken() bool {

	if w.token != nil {
		if w.token.ExpireAt > time.Now().Unix() {
			return true
		} else {
			w.saveAccessToken(nil)
		}
	}

	return false
}

func (w *WorkWeixin) GetAccessToken() string {

	if w.checkToken() {
		return w.token.Access_token
	}
	var token *AccessToken
	var buffer []byte

	token, err := w.getTokenByCache()
	if err != nil {
		log.Print(err)
		log.Print("重新通过网络获取")
		url := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/gettoken?corpid=%s&corpsecret=%s", w.corpid, w.corpsecret)
		buffer, err = GetRequestUrl(url)
	} else {
		w.token = token
		return w.token.Access_token

	}

	err = json.Unmarshal(buffer, &token)
	if err != nil {
		log.Printf("err=%+v,bufer=%v", err, buffer)
	}
	if token.Errcode != 0 {
		//return nil, errors.New(fmt.Sprintf("获取accessToken err 。 %s", token.errmsg)
		log.Fatal("获取accessToken err")
	}

	token.ExpireAt = time.Now().Unix() + token.Expires_in - 60
	buffer, _ = json.Marshal(token)
	w.saveAccessToken(buffer)
	log.Println(token)
	w.token = token

	return token.Access_token

}

func (w *WorkWeixin) getTokenByCache() (*AccessToken, error) {
	file, err := os.Open(w.getStoreFile()) // For read access.
	if err != nil {
		return nil, err
	}
	defer file.Close()

	buffer, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}
	var token *AccessToken
	_ = json.Unmarshal(buffer, &token)
	if token == nil || token.Errcode != 0 {
		log.Fatal("get cache err")
		return nil, errors.New(fmt.Sprintf("getTokenByCache 获取accessToken err"))
	}

	if token.ExpireAt > time.Now().Unix()+token.Expires_in+60 {
		return nil, errors.New(fmt.Sprintf("缓存token已经过期了 "))
	}
	return token, nil
}
func (w *WorkWeixin) getStoreFile() string {
	return fmt.Sprintf("data/work_%d.json", w.agentId)
}

func (w *WorkWeixin) saveAccessToken(bytes []byte) {

	if bytes == nil {
		_ = os.Remove(w.getStoreFile())
		return
	}
	file, err := os.Create(w.getStoreFile()) // For read access.
	if err != nil {
		log.Fatal(err)
		return
	}
	defer file.Close()
	file.Write(bytes)

}

func (w *WorkWeixin) saveGroupId(group string) {
	path := fmt.Sprintf("data/groups.json")
	file, err := os.Create(path) // For read access.
	if err != nil {
		log.Fatal(err)
		return
	}
	defer file.Close()
	w.existGroupIds = append(w.existGroupIds, group)
	resultBuf, _ := json.Marshal(w.existGroupIds)
	_, _ = file.Write(resultBuf)
}

func (w *WorkWeixin) getGroupIds() {
	path := fmt.Sprintf("data/groups.json")
	file, err := os.Open(path) // For read access.
	if err != nil {
		var groups []string
		w.existGroupIds = groups
		return
	}
	defer file.Close()
	buffer, err := ioutil.ReadAll(file)
	var groups []string
	_ = json.Unmarshal(buffer, &groups)
	w.existGroupIds = groups
}

func PostRequestUrl(url string, body io.Reader) ([]byte, error) {
	return requestUrl(url, "POST", body)
}

func GetRequestUrl(url string) ([]byte, error) {
	return requestUrl(url, "GET", nil)
}

func requestUrl(url string, method string, body io.Reader) ([]byte, error) {

	client := &http.Client{}

	request, err := http.NewRequest(method, url, body)

	if err != nil {
		return nil, err

	}

	//处理返回结果
	response, err := client.Do(request)

	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	buf, err := ioutil.ReadAll(response.Body)

	if response.StatusCode == http.StatusOK {
		return buf, nil
	} else {
		return nil, errors.New(fmt.Sprint("StatusCode=", response.StatusCode, " msg=", string(buf)))
	}

}

func (w *WorkWeixin) GetUserIdByMobile(mobile string) string {
	if w.userMobiles == nil {
		w.userMobiles = make(map[string]string)
		for _, v := range w.GetDepartmentUsers(1, 1) {
			w.userMobiles[v.Mobile] = v.Userid
		}
	}
	return w.userMobiles[mobile]
}

func (w *WorkWeixin) SendGroupText(users []string, title string, content string) string {
	chatId := w.CreateChatGroup(users, title)
	req := requestChatMarkdownMsg{
		Msgtype: "markdown",
		ChatId:  chatId,
		Content: &requestChatContent{
			Content: content,
		},
	}
	url := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/appchat/send?access_token=%s", w.token.Access_token)
	if data, err := json.Marshal(req); err == nil {
		log.Printf("发送群聊body:%s", string(data))
		if responseBuffer, e := PostRequestUrl(url, bytes.NewBuffer(data)); e == nil {
			respon := string(responseBuffer)
			log.Println(respon)
			return respon
		}
	}
	return ""
}

func (w *WorkWeixin) SendTemplateMsg(card TemplateCard) string {
	card.AgentId = w.agentId
	w.GetAccessToken()
	url := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/message/send?access_token=%s", w.token.Access_token)
	if data, err := json.Marshal(card); err == nil {
		log.Printf("发送卡片模板消息body:%s", string(data))
		if responseBuffer, e := PostRequestUrl(url, bytes.NewBuffer(data)); e == nil {
			respon := string(responseBuffer)
			log.Println(respon)
			return respon
		}
	}
	return ""
}

func (w *WorkWeixin) CreateChatGroup(users []string, title string) string {
	w.GetAccessToken()
	url := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/appchat/create?access_token=%s", w.token.Access_token)
	chatId := innerMD5(title)
	w.getGroupIds()
	for i := 0; i < len(w.existGroupIds); i++ {
		log.Printf("=========%+v\n", w.existGroupIds)
		if w.existGroupIds[i] == chatId {
			return chatId
		}
	}
	chatInfo := requestChatInfo{
		ChatId:   chatId,
		UserList: users,
		Name:     title,
		Owner:    users[0],
	}
	if marshal, err := json.Marshal(chatInfo); err == nil {
		log.Printf("request create chat body=%s\n", string(marshal))
		if buffer, e := PostRequestUrl(url, bytes.NewBuffer(marshal)); e == nil {
			log.Println(string(buffer))
			var result ResponseBase
			json.Unmarshal(buffer, &result)
			if result.Errcode == 0 {
				return chatId
			} else {
				log.Panic("创建群主失败")
			}
		} else {
			log.Println(e)
		}
	} else {
		log.Println(err)
	}
	return chatId

}

func (w *WorkWeixin) VerityCallback(signature string, timestamp string, nonce string, echostr string) ([]byte, error) {
	w.GetAccessToken()
	var wxCrypt = wxbizmsgcrypt.NewWXBizMsgCrypt(w.chatConfig.CallBackToken, w.chatConfig.CallBackEncodingAESKey, w.chatConfig.Corpid, wxbizmsgcrypt.XmlType)
	msg, cryptError := wxCrypt.VerifyURL(signature, timestamp, nonce, echostr)
	log.Println(msg)
	log.Println(cryptError)
	return msg, errors.New(fmt.Sprintf("%+v", cryptError))
}

type WechatEvent struct {
	FromUserName string `xml:"FromUserName"`
	MsgType      string `xml:"MsgType"`
	Event        string `xml:"Event"`
	EventKey     string `xml:"EventKey"`
	Content      string `xml:"Content"`
}

func (w *WorkWeixin) Callback(signature string, timestamp string, nonce string, data []byte) (WechatEvent, error) {
	w.GetAccessToken()
	var wxCrypt = wxbizmsgcrypt.NewWXBizMsgCrypt(w.chatConfig.CallBackToken, w.chatConfig.CallBackEncodingAESKey, w.chatConfig.Corpid, wxbizmsgcrypt.XmlType)
	msg, cryptError := wxCrypt.DecryptMsg(signature, timestamp, nonce, data)
	log.Println(string(msg))
	log.Println(cryptError)
	we := WechatEvent{}
	if cryptError != nil {
		return we, errors.New(fmt.Sprintf("%+v", cryptError))
	}
	xml.Unmarshal(msg, &we)

	fmt.Printf("%+v", we)

	return we, nil
}

func innerMD5(body interface{}) string {
	if data, err := json.Marshal(body); err == nil {
		has := md5.Sum(data)
		md5str1 := fmt.Sprintf("%x", has) //将[]byte转成16进制
		fmt.Println(md5str1)
		return md5str1
	} else {
		log.Fatalln(err)
	}

	return ""

}
