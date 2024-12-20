// Package symcon provides a client for the Symcon JSON-RPC API.
// https://www.symcon.de/de/service/dokumentation/entwicklerbereich/datenaustausch/
package symcon

import (
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/tommi2day/gomodules/common"

	"github.com/go-resty/resty/v2"
	log "github.com/sirupsen/logrus"
)

var httpClient *resty.Client
var debug = false

const defaultTimeout = 10
const readyState = "KR_READY"

// New creates a new Symcon object
func New(endpoint, email, password string) *Symcon {
	// set resty logger
	httpClient = resty.New()
	l := log.StandardLogger()
	if debug {
		l.SetLevel(log.DebugLevel)
	} else {
		l.SetLevel(log.ErrorLevel)
	}
	httpClient.SetDebug(debug)
	httpClient.SetLogger(l)
	return &Symcon{
		Endpoint: endpoint,
		Email:    email,
		Password: password,
		Timeout:  defaultTimeout,
		Logger:   l,
	}
}

// SetTimeout sets the timeout for the Symcon object
func (s *Symcon) SetTimeout(timeout int) {
	s.Timeout = timeout
}

// SetURL updates the URL for the Symcon Server
func (s *Symcon) SetURL(endpoint string) {
	s.Endpoint = endpoint
}

// SetEmail updates the email for the Symcon object
func (s *Symcon) SetEmail(email string) {
	s.Email = email
}

// SetPassword update the password for the Symcon object
func (s *Symcon) SetPassword(password string) {
	s.Password = password
}

// SetDebug updates the debug flag for the Symcon object
func (s *Symcon) SetDebug(d bool) {
	debug = d
	l := s.Logger
	if debug {
		l.SetLevel(log.DebugLevel)
	} else {
		l.SetLevel(log.ErrorLevel)
	}
	httpClient.SetDebug(debug)
	httpClient.SetLogger(l)
}

// QueryAPI queries the Symcon API with the given method and arguments
func (s *Symcon) QueryAPI(method string, arguments ...interface{}) (apiResponse *APIResponse, err error) {
	var resp *resty.Response
	response := new(APIResponse)
	// Create a Resty Client
	email := s.Email
	pass := s.Password
	endpoint := s.Endpoint
	// auth = encode_base64(user + ':' + pass)
	if email == "" {
		err = fmt.Errorf("no license email set")
		return
	}
	if pass == "" {
		err = fmt.Errorf("no password set")
		return
	}

	if endpoint == "" {
		err = fmt.Errorf("no endpoint set")
		return
	}

	l := log.StandardLogger()
	if debug {
		l.SetLevel(log.DebugLevel)
	} else {
		l.SetLevel(log.ErrorLevel)
	}
	// reset params
	httpClient.QueryParam = url.Values{}
	httpClient.SetHeader("Content-Type", "text/xml")
	httpClient.SetDebug(debug)
	httpClient.SetLogger(l)
	// reset params
	httpClient.QueryParam = url.Values{}
	httpClient.SetHeader("Content-Type", "application/json; charset=utf-8'")
	httpClient.SetBasicAuth(email, pass)
	httpClient.SetHeader("Accept", "application/json")
	httpClient.SetTimeout(time.Duration(s.Timeout) * time.Second)

	if len(arguments) == 0 {
		arguments = []interface{}{}
	}
	//#api function definition
	rpc := APIRequest{RPC: "2.0",
		Method: method,
		Params: arguments,
		ID:     "0",
	}
	j, err := json.Marshal(rpc)
	if err != nil {
		err = fmt.Errorf("cannot marshal request: %s", err)
		return
	}
	req := httpClient.R().SetBody(j)

	// do the request
	resp, err = req.Post(endpoint)
	if err != nil {
		err = fmt.Errorf("cannot do api request: %s", err)
		return
	}

	// check response code
	if resp.StatusCode() != 200 {
		err = fmt.Errorf("invalid status code: %d", resp.StatusCode())
		return
	}

	// check response body
	data := resp.Body()
	j = data
	log.Debugf("response body: %s", j)
	err = json.Unmarshal(data, response)
	if err != nil {
		err = fmt.Errorf("cannot unmarshal response: %s", err)
		return
	}
	if response.Error != nil && response.Error.Code != 0 {
		log.Warnf("api error: %s", response.Error)
		err = fmt.Errorf("api error: %s", response.Error.Message)
		return
	}
	if response.Result == nil {
		err = fmt.Errorf("no result returned")
		return
	}
	return response, nil
}

// CheckIPSCmdOK checks if the API response to a command is OK
func (s *Symcon) CheckIPSCmdOK(resp *APIResponse, e error) (err error) {
	if e != nil {
		err = fmt.Errorf("cannot do api request: %s", e)
		return
	}
	if resp.Result == nil {
		err = fmt.Errorf("no result returned")
		return
	}
	if resp.Error != nil {
		m := resp.Error.Message
		return fmt.Errorf("api returned %s", m)
	}
	t, ok := resp.Result.(bool)
	if ok {
		if !t {
			return fmt.Errorf("api returned false")
		}
		return nil
	}
	n, nok := resp.Result.(float64)
	if nok {
		if n == 0 {
			return fmt.Errorf("api returned zero")
		}
		return nil
	}
	return fmt.Errorf("api returned wrong type")
}

// SetIPSNameIdentParent sets the name, ident and parent of an object in the Symcon server
func (s *Symcon) SetIPSNameIdentParent(id int, name, ident string, parent int) (err error) {
	var resp *APIResponse
	if name == "" {
		err = fmt.Errorf("no name set")
		return
	}
	resp, err = s.QueryAPI("IPS_SetName", id, name)
	err = s.CheckIPSCmdOK(resp, err)
	if err != nil {
		err = fmt.Errorf("cannot set object name: %s", err)
		return
	}

	if parent > 0 {
		resp, err = s.QueryAPI("IPS_SetParent", id, parent)
		err = s.CheckIPSCmdOK(resp, err)
		if err != nil {
			err = fmt.Errorf("cannot set object parent: %s", err)
			return
		}
	}

	if ident != "" {
		resp, err = s.QueryAPI("IPS_SetIdent", id, ident)
		err = s.CheckIPSCmdOK(resp, err)
		if err != nil {
			err = fmt.Errorf("cannot set object ident: %s", err)
			return
		}
	}

	return
}

// SetIPSVarValue sets the value of a variable in the Symcon server
func (s *Symcon) SetIPSVarValue(id int, value interface{}) (err error) {
	var resp *APIResponse
	var v *IPSVariable
	var i interface{}
	v, err = s.GetIPSVariableInfo(id)
	if err != nil {
		err = fmt.Errorf("cannot check variable exists: %s", err)
		return
	}
	t := v.VariableType
	switch t {
	case 0: // boolean
		if _, ok := value.(bool); !ok {
			err = fmt.Errorf("value is not a boolean")
			return
		}
		i = value
	case 1: // integer
		if _, ok := value.(int64); !ok {
			err = fmt.Errorf("value is not an integer")
			return
		}
		i = value
	case 2: // float
		if _, ok := value.(float64); !ok {
			err = fmt.Errorf("value is not a float")
			return
		}
		i = value
	case 3: // string
		if _, ok := value.(string); !ok {
			err = fmt.Errorf("value is not a string")
			return
		}
		i = value
	case 4: // array
		if _, ok := value.([]interface{}); !ok {
			err = fmt.Errorf("value is not an array")
			return
		}
		i = value
	case 5: // variant
		i = value
	}

	resp, err = s.QueryAPI("SetValue", id, i)
	err = s.CheckIPSCmdOK(resp, err)
	if err != nil {
		err = fmt.Errorf("cannot set value: %s", err)
		return
	}
	return
}

// IPSVariableExists checks if a variable exists in the Symcon server
func (s *Symcon) IPSVariableExists(id int) (exists bool, err error) {
	var resp *APIResponse
	resp, err = s.QueryAPI("IPS_VariableExists", id)
	if err != nil {
		err = fmt.Errorf("cannot check variable exists: %s", err)
		return false, err
	}
	exists = resp.Result.(bool)
	log.Debugf("variable exists: %v", exists)
	return exists, nil
}

// GetIPSObject returns the IPS object for a given ID
func (s *Symcon) GetIPSObject(id int) (ipsObject *IPSObject, err error) {
	var resp *APIResponse
	resp, err = s.QueryAPI("IPS_GetObject", id)
	if err != nil {
		err = fmt.Errorf("cannot get object %d: %s", id, err)
		return
	}
	r := resp.Result.(map[string]interface{})
	if len(r) == 0 {
		err = fmt.Errorf("no object properties returned")
		return nil, err
	}
	o := IPSObject{}
	o.ParentID = int(r["ParentID"].(float64))
	o.ObjectID = int(r["ObjectID"].(float64))
	o.ObjectType = int(r["ObjectType"].(float64))
	o.ObjectIdent = r["ObjectIdent"].(string)
	o.ObjectName = r["ObjectName"].(string)
	o.ObjectInfo = r["ObjectInfo"].(string)
	o.ObjectIcon = r["ObjectIcon"].(string)
	o.ObjectSummary = r["ObjectSummary"].(string)
	o.ObjectPosition = int(r["ObjectPosition"].(float64))
	o.ObjectIsReadOnly = r["ObjectIsReadOnly"].(bool)
	o.ObjectIsHidden = r["ObjectIsHidden"].(bool)
	o.ObjectIsDisabled = r["ObjectIsDisabled"].(bool)
	o.ObjectIsLocked = r["ObjectIsLocked"].(bool)
	o.HasChildren = r["HasChildren"].(bool)
	o.ChildrenIDs = make([]int, 0)
	for _, v := range r["ChildrenIDs"].([]interface{}) {
		o.ChildrenIDs = append(o.ChildrenIDs, int(v.(float64)))
	}
	p, err := s.GetObjectPath(id)
	if err != nil {
		err = fmt.Errorf("cannot get object path: %s", err)
		return nil, err
	}
	o.ObjectPath = p
	return &o, nil
}
func (s *Symcon) GetIPSVariableInfo(id int) (ipsVariable *IPSVariable, err error) {
	var obj *IPSObject
	var varResp *APIResponse
	var valResp *APIResponse
	var profileResp *APIResponse
	exists, err := s.IPSVariableExists(id)
	if err != nil {
		err = fmt.Errorf("cannot check variable exists: %s", err)
		return
	}
	if !exists {
		err = fmt.Errorf("variable %d doesnt exist", id)
		return
	}
	obj, err = s.GetIPSObject(id)
	log.Debugf("object: %v", obj)
	if err != nil {
		err = fmt.Errorf(" getobject id %d failed: %s", id, err)
		return
	}
	name := obj.ObjectName
	objType := obj.ObjectType
	ident := obj.ObjectIdent
	parent := obj.ParentID
	if objType != 2 {
		err = fmt.Errorf("not a variable type: %d", objType)
		return
	}
	varResp, err = s.QueryAPI("IPS_GetVariable", id)
	log.Debugf("variable response: %v", varResp)
	if err != nil {
		err = fmt.Errorf("cannot get variable %d: %s", id, err)
		return
	}
	variable := varResp.Result.(map[string]interface{})
	if len(variable) == 0 {
		err = fmt.Errorf("no variable properties returned")
		return
	}
	v := IPSVariable{}
	v.Name = name
	v.Ident = ident
	v.Parent = parent
	v.VariableID = int(variable["VariableID"].(float64))
	v.VariableType = int(variable["VariableType"].(float64))
	v.VariableUpdated = int64(variable["VariableUpdated"].(float64))
	v.VariablePath = obj.ObjectPath
	vp := variable["VariableCustomProfile"].(string)
	if vp == "" {
		vp = variable["VariableProfile"].(string)
	}
	v.VariableProfileName = vp
	// get variable profile if name is set
	if vp != "" {
		profileResp, err = s.QueryAPI("IPS_GetVariableProfile", vp)
		if err != nil {
			err = fmt.Errorf("cannot get variable profile %s: %s", vp, err)
			return
		}
		profile := profileResp.Result.(map[string]interface{})
		log.Debugf("profile response: %v", profile)
		if len(profile) == 0 {
			err = fmt.Errorf("no variable profile properties returned")
			return
		}
		p := IPSVariableProfile{}
		p.ProfileName = profile["ProfileName"].(string)
		p.ProfileType = int(profile["ProfileType"].(float64))
		p.Digits = int(profile["Digits"].(float64))
		p.Icon = profile["Icon"].(string)
		p.IsReadOnly = profile["IsReadOnly"].(bool)
		p.MaxValue = profile["MaxValue"].(float64)
		p.MinValue = profile["MinValue"].(float64)
		p.Prefix = profile["Prefix"].(string)
		p.StepSize = profile["StepSize"].(float64)
		p.Suffix = profile["Suffix"].(string)
		log.Debugf("Profile added: %s", p.ProfileName)
		if profile["Associations"] != nil {
			a := make([]IPSVariableAssociation, 0)
			for _, v := range profile["Associations"].([]interface{}) {
				assoc := IPSVariableAssociation{}
				assoc.Value = v.(map[string]interface{})["Value"].(float64)
				assoc.Name = v.(map[string]interface{})["Name"].(string)
				assoc.Icon = v.(map[string]interface{})["Icon"].(string)
				assoc.Color = int(v.(map[string]interface{})["Color"].(float64))
				a = append(a, assoc)
				log.Debugf("association defined: %s", assoc)
			}
			v.VariableAssociations = &a
			log.Debugf("associations added: %d to profile %s", len(*v.VariableAssociations), p.ProfileName)
		}
		v.VariableProfile = &p
	}

	// get variable value
	valResp, err = s.QueryAPI("GetValue", id)
	if err != nil {
		err = fmt.Errorf("cannot get variable value: %s", err)
		return
	}
	switch v.VariableType {
	case 0: // boolean
		v.Value = valResp.Result.(bool)
	case 1: // integer comes as float
		v.Value = int64(valResp.Result.(float64))
	case 2: // float
		v.Value = valResp.Result.(float64)
	case 3: // string
		v.Value = valResp.Result.(string)
	case 4: // array
		v.Value = valResp.Result.([]interface{})
	case 5: // variant
		v.Value = valResp.Result
	}
	log.Debugf("variable result: %v", v)
	return &v, nil
}

// GetObjectPath returns the path of an object in the Symcon server
func (s *Symcon) GetObjectPath(id int) (path string, err error) {
	var resp *APIResponse
	path = ""
	if id == 0 {
		return
	}
	for id > 0 {
		// get parent
		resp, err = s.QueryAPI("IPS_GetParent", id)
		if err != nil || resp == nil {
			log.Debugf("IPS_GetParent failed for id: %d", id)
			path = ""
			return
		}
		parent := int(resp.Result.(float64))
		log.Debugf("parent for ID %d : %d", id, parent)

		// find actual name
		resp, err = s.QueryAPI("IPS_GetName", id)
		if err != nil || resp == nil {
			log.Debugf("IPS_GetName failed for id: %d", id)
			path = ""
			return
		}
		name := resp.Result.(string)
		if name == "" {
			name = fmt.Sprintf("id-%d", id)
		}
		log.Debugf("name for ID %d : %s", id, name)
		path = name + "/" + path
		id = parent
	}
	path = path[:len(path)-1]
	log.Debugf("Symcon path: %s", path)

	return
}

// IsReady checks if the Symcon Kernel is ready
func (s *Symcon) IsReady() bool {
	rl := common.ReverseMap(IPSKernelRunlevel)
	stReady := rl[readyState]
	ready := false
	resp, err := s.QueryAPI("IPS_GetKernelRunlevel")
	if err == nil && resp != nil {
		status := int(resp.Result.(float64))
		ready = status == stReady
	}
	return ready
}

func (v *IPSVariable) String() string {
	assoc := "None"
	profile := ""
	if v == nil {
		return "No variable data"
	}
	if v.VariableProfile == nil {
		profile = "No profile"
	} else {
		profile = v.VariableProfile.ProfileName
	}
	if v.VariableAssociations != nil {
		a := *v.VariableAssociations
		if len(a) > 0 {
			assoc = ""
			for i, v := range a {
				assoc += fmt.Sprintf("\nAssociation %d: %s", i, v)
			}
		}
	}

	return fmt.Sprintf("ID: %d, Name: %s, Type: %s, Updated: %s, Value: %v, Profile: %s, Associations: %s",
		v.VariableID, v.Name, IPSVarTypes[v.VariableType], time.Unix(v.VariableUpdated, 0).String(), v.Value, profile, assoc)
}
func (r *APIResponse) String() string {
	if r == nil {
		return "No response data"
	}
	return fmt.Sprintf("RPC: %s, ID: %s, Result: %v, Error: %v",
		r.RPC, r.ID, r.Result, r.Error)
}
func (r *APIError) String() string {
	if r == nil {
		return "No error"
	}
	return fmt.Sprintf("Code: %d, Message: %s",
		r.Code, r.Message)
}
func (r *APIRequest) String() string {
	if r == nil {
		return "No request data"
	}
	return fmt.Sprintf("RPC: %s, Method: %s, Params: %v, ID: %s",
		r.RPC, r.Method, r.Params, r.ID)
}
func (a IPSVariableAssociation) String() string {
	return fmt.Sprintf("Name: %s, Value: %f, Icon: %s, Color: %d",
		a.Name, a.Value, a.Icon, a.Color)
}

func (o *IPSObject) String() string {
	if o == nil {
		return "No object data"
	}
	return fmt.Sprintf("ID: %d, Name: %s, Type: %s, Ident: %s, Parent: %d, Path: %s",
		o.ObjectID, o.ObjectName, IPSObjectTypes[o.ObjectType], o.ObjectIdent, o.ParentID, o.ObjectPath)
}
