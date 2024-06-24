package symcon

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tommi2day/gomodules/common"
	"github.com/tommi2day/gomodules/test"
)

var testURL = ipsURL
var varTypesReverse = common.ReverseMap(IPSVarTypes)
var objTypesReverse = common.ReverseMap(IPSObjectTypes)
var hexBlue, _ = common.GetHexInt64Val("0000FF")
var hexRed, _ = common.GetHexInt64Val("FF0000")

const testUser = "test@symcon.de"
const testPass = "ipsymcon"
const testVariableName = "TestVariable"
const testVariableIdent = "test"
const testCategoryName = "TestCategory"
const testCategoryIdent = "testcat"
const testVariableValue = 20.1
const testVariableProfile = "MyTemperature"

var testVariableType = varTypesReverse["float64"]

func TestSymcon(t *testing.T) {
	var resp *APIResponse
	if os.Getenv("SKIP_IPS") != "" {
		t.Skip("Skipping IPS testing in CI environment")
	}
	test.InitTestDirs()

	ipsContainer, err := prepareIpsContainer()
	require.NoErrorf(t, err, "IPS Server not available")
	require.NotNil(t, ipsContainer, "Prepare failed")
	defer common.DestroyDockerContainer(ipsContainer)
	t.Run("Test noPassword", func(t *testing.T) {
		ips := New(testURL, "wronguser", "")
		resp, err = ips.QueryAPI("GetVariable", 0)
		assert.Error(t, err, "QueryAPI should return an error")
		assert.Nil(t, resp, "QueryAPI should not return a response")
		t.Log(err)
	})
	ips := New("", "", "")
	t.Run("Test noURL", func(t *testing.T) {
		resp, err = ips.QueryAPI("GetVariable", 0)
		assert.Error(t, err, "QueryAPI should return an error")
		assert.Nil(t, resp, "QueryAPI should not return a response")
		t.Log(err)
	})
	ips.SetEmail(testUser)
	ips.SetPassword(testPass)
	ips.SetURL(testURL)
	ips.SetDebug(true)
	ready := false
	t.Run("Test IPS Ready", func(t *testing.T) {
		kMap := common.ReverseMap(IPSKernelRunlevel)
		e := kMap["KR_READY"]
		resp, err = ips.QueryAPI("IPS_GetKernelRunlevel")
		assert.NoErrorf(t, err, "QueryAPI should not return an error: %v", err)
		assert.NotNil(t, resp, "QueryAPI should not return a response")
		if resp != nil {
			status := int(resp.Result.(float64))
			assert.Equal(t, e, status, "IPS should be ready")
			ready = e == status
		}
	})
	if !ready {
		t.Fatalf("IPS not ready")
	}
	varid := 0
	catid := 0
	t.Run("Test CreateCategory", func(t *testing.T) {
		resp, err = ips.QueryAPI("IPS_CreateCategory")
		err = ips.CheckIPSCmdOK(resp, err)
		assert.NoErrorf(t, err, "CreateObject should not return an error:%s", err)
		assert.NotNil(t, resp, "QueryAPI should return a response")
		if resp != nil {
			i := int(resp.Result.(float64))
			catid = i
			err = ips.SetIPSNameIdentParent(catid, testCategoryName, testCategoryIdent, 0)
			assert.NoErrorf(t, err, "SetIPSNameIdentParent should not return an error:%s", err)
		}
	})
	if catid == 0 {
		t.Fatalf("Category not created")
	}
	t.Run("Test GetObject", func(t *testing.T) {
		var obj *IPSObject
		obj, err = ips.GetIPSObject(catid)
		assert.NoError(t, err, "GetIPSObject should not return an error:%s", err)
		assert.NotNil(t, obj, "GetIPSObject should return an object")
		if obj != nil {
			assert.Equal(t, testCategoryName, obj.ObjectName, "Category name should not be empty")
			assert.Equal(t, testCategoryIdent, obj.ObjectIdent, "Category ident should be test")
			assert.Equal(t, 0, obj.ParentID, "Category parent should be root")
			assert.Equal(t, objTypesReverse["category"], obj.ObjectType, "Category type should be Category")
			t.Logf("Category: %v", obj)
		}
	})
	if err != nil {
		t.Fatalf("Access to Symcon failed: %v", err)
	}
	t.Run("Test Create Profile", func(t *testing.T) {
		resp, err = ips.QueryAPI("IPS_CreateVariableProfile", testVariableProfile, testVariableType)
		err = ips.CheckIPSCmdOK(resp, err)
		assert.NoErrorf(t, err, "CreateObject should not return an error:%s", err)
		assert.NotNil(t, resp, "QueryAPI should return a response")
		resp, err = ips.QueryAPI("IPS_SetVariableProfileAssociation", testVariableProfile, 0, "cold", "", hexBlue)
		err = ips.CheckIPSCmdOK(resp, err)
		assert.NoErrorf(t, err, "SetVariableProfileAssociation Blue should not return an error:%s", err)
		resp, err = ips.QueryAPI("IPS_SetVariableProfileAssociation", testVariableProfile, 40, "hot", "sun", hexRed)
		err = ips.CheckIPSCmdOK(resp, err)
		assert.NoErrorf(t, err, "SetVariableProfileAssociation Red should not return an error:%s", err)
	})
	t.Run("Test CreateVariable", func(t *testing.T) {
		resp, err = ips.QueryAPI("IPS_CreateVariable", testVariableType)
		err = ips.CheckIPSCmdOK(resp, err)
		assert.NoErrorf(t, err, "CreateObject should not return an error:%s", err)
		assert.NotNil(t, resp, "QueryAPI should return a response")
		if resp != nil {
			i := int(resp.Result.(float64))
			varid = i
			assert.NotZero(t, varid, "CreateVariable should return an ID")
			err = ips.SetIPSNameIdentParent(varid, testVariableName, testVariableIdent, catid)
			assert.NoErrorf(t, err, "SetIPSNameIdentParent should not return an error:%s", err)
			resp, err = ips.QueryAPI("IPS_SetVariableCustomProfile", varid, testVariableProfile)
			err = ips.CheckIPSCmdOK(resp, err)
			assert.NoErrorf(t, err, "SetVariableCustomProfile should not return an error:%s", err)
		}
	})
	if varid == 0 {
		t.Fatalf("Variable not created")
	}
	t.Run("Test Variable Exists", func(t *testing.T) {
		exists := false
		exists, err = ips.IPSVariableExists(varid)
		assert.NoErrorf(t, err, "QueryAPI should not return an error:%s", err)
		assert.True(t, exists, "Variable should exist")
	})
	t.Run("Test SetIPSVarValue", func(t *testing.T) {
		err = ips.SetIPSVarValue(varid, testVariableValue)
		assert.NoErrorf(t, err, "SetIPSVariableValue should not return an error:%s", err)
	})
	t.Run("Test GetVariable", func(t *testing.T) {
		variable, err := ips.GetIPSVariableInfo(varid)
		assert.NoErrorf(t, err, "GetVariable should not return an error:%s", err)
		if variable != nil {
			assert.Equal(t, testVariableType, variable.VariableType, "Variable type should be float64")
			assert.Equalf(t, testVariableName, variable.Name, "Variable name should  be %s", testVariableName)
			assert.Equalf(t, varid, variable.VariableID, "Variable ID should not be %d", varid)
			assert.Equalf(t, testVariableValue, variable.Value, "Variable value should be %f", testVariableValue)
			assert.Equalf(t, testVariableIdent, variable.Ident, "Variable ident should be %s", testVariableIdent)
			assert.Equalf(t, catid, variable.Parent, "Variable parent should be category id %d", catid)
			t.Logf("Variable: %v", variable)
		}
	})
	t.Run("Test Object path", func(t *testing.T) {
		path := ""
		path, err = ips.GetObjectPath(varid)
		assert.NoErrorf(t, err, "GetObjectPath should not return an error:%s", err)
		assert.NotEmpty(t, path, "Object path should not be empty")
		assert.Equal(t, testCategoryName+"/"+testVariableName, path, "Object path should be category/variable")
		t.Logf("Object path: %s", path)
	})
}
