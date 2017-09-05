package work

import (
	"fmt"
	"encoding/json"
	"time"
	"log"
	"errors"
	"os"
	"io"
	"io/ioutil"
	"net/http"
	"bytes"
)

type WorkWeixin struct {
	corpid     string
	corpsecret string
	token      *AccessToken
	agentId    int
}

type AccessToken struct {
	Access_token string `json:"access_token"`
	Expires_in   int64  `json:"expires_in"`  //秒 默认返回是2小时
	ExpireAt     int64     `json:"expireAt"` //在什么时候过期
	Errcode      int `json:"errcode"`
	Errmsg       string `json:"errmsg"`
}

type Department struct {
	Id       int `json:"id"`
	Name     string `json:"name"`
	Parentid int32 `json:"parentid"`
	Order    int32 `json:"order"`
}

type departments struct {
	Errcode    int `json:"errcode"`
	Errmsg     string `json:"errmsg"`
	Department []Department `json:"department"`
}

type users struct {
	Errcode  int `json:"errcode"`
	Errmsg   string `json:"errmsg"`
	Userlist []User `json:"userlist"`
}

type User struct {
	Userid       string `json:"userid"`
	Name         string `json:"name"`
	Department   []int `json:"department"`
	Mobile       string `json:"mobile"`
	Email        string `json:"email"`
	Status       int `json:"status"`
	Avatar       string `json:"avatar"`
	Telephone    string `json:"telephone"`
	English_name string `json:"english_name"`
}

//agentId 表示应用id， 0  表示本应用
//微信
func (w *WorkWeixin) Init(corpid string,
	corpsecret string, agentId int) {
	w.corpid = corpid
	w.corpsecret = corpsecret
	w.agentId = agentId;
	w.GetAccessToken()

}

type Tag struct {
	TagName string `json:"tagname"`
	TagId   int `json:"tagid"`
}
type tags struct {
	Errcode int `json:"errcode"`
	Errmsg  string `json:"errmsg"`
	TagList []Tag `json:"taglist"`
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
	buffer, err := getRequestUrl(url)
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
	buffer, err := getRequestUrl(url)
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
	buffer, err := postRequestUrl(url, bytes.NewBuffer(body))
	if err != nil {
		log.Panic(err)
	}
	log.Println("CreateTag Unmarshal err=", err, string(buffer))

}

//获取标签
func (w *WorkWeixin) GetTagList() []Tag {

	url := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/tag/list?access_token=%s",
		w.GetAccessToken())

	buffer, err := getRequestUrl(url)
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

	buffer, err := getRequestUrl(url)
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
	buffer, err := postRequestUrl(url, bytes.NewBuffer(body))

	if err != nil {
		log.Panic(err)
	}

	return string(buffer)
}

func (w *WorkWeixin) SendText(toUser string, toparties string, totag string, text string) {

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
		return
	}
	log.Println(string(body))

	buffer, err := postRequestUrl(fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/message/send?access_token=%s",
		w.GetAccessToken()),
		bytes.NewBuffer(body))
	//
	if err != nil {
		log.Println("send text err", err)
	} else {
		log.Println(string(buffer))
	}

}

func (w *WorkWeixin) checkToken() bool {
	return w.token != nil && w.token.ExpireAt > time.Now().Unix()
}

func (w *WorkWeixin) GetAccessToken() (string) {

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
		buffer, err = getRequestUrl(url)
	} else {
		w.token = token
		return w.token.Access_token

	}

	json.Unmarshal(buffer, &token)
	if token.Errcode != 0 {
		//return nil, errors.New(fmt.Sprintf("获取accessToken err 。 %s", token.errmsg)
		log.Fatal("获取accessToken err")
	}

	w.saveAccessToken(buffer)
	token.ExpireAt = time.Now().Unix() + token.Expires_in - 60
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
	json.Unmarshal(buffer, &token)
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
	return fmt.Sprintf("/data/work_%d.json", w.agentId)
}

func (w *WorkWeixin) saveAccessToken(bytes []byte) {

	file, err := os.Create(w.getStoreFile()) // For read access.
	if err != nil {
		log.Fatal(err)
		return
	}
	defer file.Close()
	file.Write(bytes)

}

func postRequestUrl(url string, body io.Reader) ([]byte, error) {
	return requestUrl(url, "POST", body)
}

func getRequestUrl(url string) ([]byte, error) {
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
