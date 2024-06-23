package symcon

import (
	"github.com/sirupsen/logrus"
)

// Symcon is a struct that holds the configuration for the Symcon object
type Symcon struct {
	Endpoint string
	Email    string
	Password string
	Timeout  int
	Logger   *logrus.Logger
}

// APIRequest is a struct that holds the request to the Symcon server
type APIRequest struct {
	RPC    string        `json:"jsonrpc"`
	Method string        `json:"method"`
	Params []interface{} `json:"params"`
	ID     string        `json:"id"`
}

type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// APIResponse is a struct that holds the response from the Symcon server
type APIResponse struct {
	RPC    string      `json:"jsonrpc"`
	ID     string      `json:"id"`
	Result interface{} `json:"result"`
	Error  *APIError   `json:"error"`
}

// IPSObject is a struct that holds the IPS object properties
type IPSObject struct {
	ParentID         int    `json:"ParentID"`
	ObjectID         int    `json:"ObjectID"`
	ObjectType       int    `json:"ObjectType"`
	ObjectIdent      string `json:"ObjectIdent"`
	ObjectName       string `json:"ObjectName"`
	ObjectInfo       string `json:"ObjectInfo"`
	ObjectIcon       string `json:"ObjectIcon"`
	ObjectSummary    string `json:"ObjectSummary"`
	ObjectPosition   int    `json:"ObjectPosition"`
	ObjectIsReadOnly bool   `json:"ObjectIsReadOnly"`
	ObjectIsHidden   bool   `json:"ObjectIsHidden"`
	ObjectIsDisabled bool   `json:"ObjectIsDisabled"`
	ObjectIsLocked   bool   `json:"ObjectIsLocked"`
	HasChildren      bool   `json:"HasChildren"`
	ChildrenIDs      []int  `json:"ChildrenIDs"`
	ObjectPath       string `json:"ObjectPath"`
}

// IPSVariable is a struct that holds the IPS variable properties
type IPSVariable struct {
	VariableID           int                       `json:"VariableID"`
	VariableType         int                       `json:"VariableType"`
	VariableUpdated      int64                     `json:"VariableUpdated"`
	VariableIsLocked     bool                      `json:"VariableIsLocked"`
	VariableCustomAction int                       `json:"VariableCustomAction"`
	Value                interface{}               `json:"Value"`
	Name                 string                    `json:"Name"`
	Ident                string                    `json:"Ident"`
	Parent               int                       `json:"Parent"`
	VariableProfileName  string                    `json:"VariableProfileName"`
	VariableProfile      *IPSVariableProfile       `json:"VariableProfile"`
	VariableAssociations *[]IPSVariableAssociation `json:"VariableAssociations"`
	VariablePath         string                    `json:"VariablePath"`
}

// IPSVariableProfile is a struct that holds the IPS variable profile properties
type IPSVariableProfile struct {
	ProfileName  string        `json:"ProfileName"`
	ProfileType  int           `json:"ProfileType"`
	Associations []interface{} `json:"Associations"`
	Digits       int           `json:"Digits"`
	Icon         string        `json:"Icon"`
	IsReadOnly   bool          `json:"IsReadOnly"`
	MaxValue     float64       `json:"MaxValue"`
	MinValue     float64       `json:"MinValue"`
	Prefix       string        `json:"Prefix"`
	StepSize     float64       `json:"StepSize"`
	Suffix       string        `json:"Suffix"`
}

// IPSVariableAssociation is a struct that holds the IPS variable association properties
type IPSVariableAssociation struct {
	Value float64 `json:"Value"`
	Name  string  `json:"Name"`
	Icon  string  `json:"Icon"`
	Color int     `json:"Color"`
}

// IPSVarTypes is a map of IPS variable types
var IPSVarTypes = map[int]string{
	0: "bool",
	1: "int64",
	2: "float64",
	3: "string",
	4: "[]interface{}",
	5: "interface{}",
}

// IPSObjectTypes is a map of IPS object types
var IPSObjectTypes = map[int]string{
	0: "category",
	1: "instance",
	2: "variable",
	3: "script",
	4: "event",
	5: "media",
	6: "link",
}

// IPSKernelRunlevel is a map of IPS kernel states
var IPSKernelRunlevel = map[int]string{
	/*
		KR_CREATE	10101	Kernel wird erstellt
		KR_INIT	    10102	Kernel wird initialisiert. z.B. werden Module geladen und Instanzen erstellt
		KR_READY	10103	Kernel ist bereit und läuft
		KR_UNINIT	10104	Kernel wird heruntergefahren und alles sauber beendet
		KR_SHUTDOWN	10105	Kernel wurde vollständig beendet
	*/
	10101: "KR_CREATE",
	10102: "KR_INIT",
	10103: "KR_READY",
	10104: "KR_UNINIT",
	10105: "KR_SHUTDOWN",
}
