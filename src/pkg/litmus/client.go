package client

import (
	"fmt"
	kubernetes "init-agent/pkg/k8s"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/buger/jsonparser"
	ymlToJson "github.com/ghodss/yaml"
	"github.com/golang-jwt/jwt"
	"github.com/litmuschaos/litmusctl/pkg/apis"
	types "github.com/litmuschaos/litmusctl/pkg/types"
	"github.com/litmuschaos/litmusctl/pkg/utils"
)

func prepareNewAgent() (types.Agent, error) {
	var err error
	var newAgent types.Agent
	newAgent.AgentName = os.Getenv("AGENT_NAME")
	newAgent.Namespace = os.Getenv("NAMESPACE")
	newAgent.Description = os.Getenv("AGENT_DESCRIPTION")
	newAgent.ProjectId = os.Getenv("LITMUS_PROJECT_ID")
	newAgent.Mode = os.Getenv("AGENT_MODE")
	newAgent.SkipSSL = true

	// -- OPTIONNAL -- //
	newAgent.ClusterType = os.Getenv("CLUSTER_TYPE")
	newAgent.NodeSelector = os.Getenv("AGENT_NODE_SELECTOR")
	newAgent.PlatformName = os.Getenv("PLATFORM_NAME")
	newAgent.ServiceAccount = os.Getenv("SERVICE_ACCOUNT_NAME")
	newAgent.SAExists, _ = strconv.ParseBool(os.Getenv("SA_EXISTS"))
	newAgent.NsExists, _ = strconv.ParseBool(os.Getenv("NS_EXISTS"))
	return newAgent, err
}

func prepareAgentConfigMap() map[string]string {
	configMapData := make(map[string]string)
	configMapData["SERVER_ADDR"] = os.Getenv("LITMUS_BACKEND_URL")
	configMapData["VERSION"] = os.Getenv("APP_VERSION")
	configMapData["IS_CLUSTER_CONFIRMED"] = "false"
	configMapData["START_TIME"] = strconv.FormatInt(time.Now().Unix(), 10)
	selector := `["litmuschaos.io/app=chaos-exporter", "litmuschaos.io/app=chaos-operator", "litmuschaos.io/app=event-tracker", "litmuschaos.io/app=workflow-controller"]`
	configMapData["COMPONENTS"] = "DEPLOYMENTS: " + selector
	configMapData["AGENT_SCOPE"] = os.Getenv("AGENT_MODE")
	configMapData["SKIP_SSL_VERIFY"] = "true"
	return configMapData
}

func prepareAgentSecret(agentConnect apis.AgentConnect, accessKey string) map[string][]byte {
	secretData := make(map[string][]byte)
	clusterID := agentConnect.UserAgentReg.ClusterID
	secretData["CLUSTER_ID"] = []byte(clusterID)
	secretData["ACCESS_KEY"] = []byte(accessKey)

	//secretData["ACCESS_KEY"] = []byte(agentConnect.UserAgentReg.Token)
	return secretData
}

func prepareWorkflowControllerConfigMap(clusterID string) map[string]string {
	configMapWorkflowController := make(map[string]string)
	configMapWorkflowController["config"] = (`    containerRuntimeExecutor: ` + os.Getenv("CONTAINER_RUNTIME_EXECUTOR") + `
    executor:
      imagePullPolicy: IfNotPresent
    instanceID: ` + clusterID)
	return configMapWorkflowController

}

func GetProjectID(credentials types.Credentials) string {
	var result string
	userDetails, err := apis.GetProjectDetails(credentials)
	if err != nil {
		fmt.Printf("Error, cannot get project details: " + err.Error())
		os.Exit(1)
	}
	for _, project := range userDetails.Data.Projects {
		for _, member := range project.Members {
			if (member.UserID == userDetails.Data.ID) && (member.Role == "Owner" || member.Role == "Editor") {
				result = project.ID
			}
		}
	}
	return result

}

func GetAgentWithName(credentials types.Credentials, searchAgent types.Agent) (apis.AgentDetails, error) {
	agents, err := apis.GetAgentList(credentials, searchAgent.ProjectId)
	if err != nil {
		return apis.AgentDetails{}, err
	}
	for _, agent := range agents.Data.GetAgent {
		if agent.AgentName == searchAgent.AgentName {
			return agent, nil
		}
	}
	return apis.AgentDetails{}, nil
}

func CreateAgent(credentials types.Credentials) {
	newAgent, err := prepareNewAgent()
	if err != nil {
		fmt.Printf("Error, cannot create agent: " + err.Error())
		os.Exit(1)
	}

	if newAgent.ProjectId == "" {
		newAgent.ProjectId = GetProjectID(credentials)
	}

	agentExist, err := GetAgentWithName(credentials, newAgent)
	if err != nil {
		fmt.Printf("Error, cannot search if agent exist: " + err.Error())
		os.Exit(1)
	}

	if (agentExist == apis.AgentDetails{}) {
		connectionData, err := apis.ConnectAgent(newAgent, credentials)
		if err != nil {
			fmt.Printf("Error, cannot declare agent. Error: " + err.Error() + "\n")
			os.Exit(1)
		}
		if (connectionData.Data == apis.AgentConnect{}) {
			fmt.Printf("❌ Agent empty: Registration failed did graphql change ? \n")
			os.Exit(1)
		}
		accessKey, err := validateAgent(connectionData.Data.UserAgentReg.Token, credentials.Endpoint)
		if err != nil {
			utils.Red.Println("❌ Error validate agent: ", err)
			os.Exit(1)
		}

		configMap := prepareAgentConfigMap()
		kubernetes.CreateConfigMap(os.Getenv("AGENT_CONFIGMAP_NAME"), configMap, os.Getenv("NAMESPACE"))

		secret := prepareAgentSecret(connectionData.Data, accessKey)
		kubernetes.CreateSecret(os.Getenv("AGENT_SECRET_NAME"), secret, os.Getenv("NAMESPACE"))

		workflowConfigMap := prepareWorkflowControllerConfigMap(connectionData.Data.UserAgentReg.ClusterID)
		kubernetes.CreateConfigMap(os.Getenv("WORKFLOW_CONTROLER_CONFIGMAP_NAME"), workflowConfigMap, os.Getenv("NAMESPACE"))

		fmt.Printf("Agent Successfully declared, starting...\n")
	} else {
		fmt.Printf("Agent already exist, starting...\n")
	}
}

func validateAgent(token string, endpoint string) (string, error) {
	var accessKey string

	path := fmt.Sprintf("%s/%s/%s.yaml", endpoint, utils.ChaosYamlPath, token)
	req, err := http.NewRequest("GET", path, nil)
	if err != nil {
		return accessKey, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return accessKey, err
	}
	defer resp.Body.Close()
	resp_body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return accessKey, err
	}
	manifests := strings.Split(string(resp_body), "---")
	for _, manifest := range manifests {
		if len(strings.TrimSpace(manifest)) > 0 {
			jsonValue, err := ymlToJson.YAMLToJSON([]byte(manifest))
			if err != nil {
				return accessKey, err
			}
			fieldName, _, _, err := jsonparser.Get([]byte(jsonValue), "metadata", "name")
			if err != nil {
				return accessKey, err
			}
			fieldKind, _, _, err := jsonparser.Get([]byte(jsonValue), "kind")
			if err != nil {
				return accessKey, err
			}
			if string(fieldName) == "agent-secret" && string(fieldKind) == "Secret" {
				if fieldData, _, _, err := jsonparser.Get([]byte(jsonValue), "stringData", "ACCESS_KEY"); err != nil {
					return accessKey, err
				} else {
					accessKey = string(fieldData)
				}
			}
		}
	}
	return accessKey, err
}

func DeleteAgent(credentials types.Credentials) {
	_ = credentials
	fmt.Println("Eeeeh aurevoir")
	// // projectID := os.Getenv("LITMUS_PROJECT_ID")
	// clusterID := os.Getenv("CLUSTER_ID")

	// utils.White_B.Println("\n🚀 Delete cluster!! 🎉")

	// //query := `{"query":"mutation {\n  deleteClusterReg(clusterInput: \n    { \n    clusterID: \"` + CLUSTER_ID + `\",\n  }){ clusterID\n }\n}"}`
	// query := `{"operationName":"deleteCluster","variables":{"clusterID":"` + clusterID + `"},"query":"mutation deleteCluster($clusterID: String\u0021) {\\n  deleteClusterReg(clusterID: $clusterID)\\n}\\n"}`
	// params := apis.SendRequestParams{Endpoint: LITMUS_FRONTEND_URL + "/api/query", Token: credentials.Token}
	// resp, err := apis.SendRequest(params, []byte(query), string(types.Post))
	// if err != nil {
	// 	utils.Red.Println("Error in getting agent list: ", err)
	// 	os.Exit(1)
	// }

	// bodyBytes, err := ioutil.ReadAll(resp.Body)
	// defer resp.Body.Close()
	// if err != nil {
	// 	utils.Red.Println("Error in getting agent list: ", err)
	// 	os.Exit(1)
	// }
	// _ = bodyBytes
	// utils.White_B.Println("\n🚀 Agent deleted Successful!! 🎉")
}

func Login(LITMUS_FRONTEND_URL string, LITMUS_USERNAME string, LITMUS_PASSWORD string) types.Credentials {
	msg := ""

	if len(LITMUS_FRONTEND_URL) == 0 {
		msg = msg + "LITMUS_FRONTEND_URL, "
	}

	if len(LITMUS_USERNAME) == 0 {
		msg = msg + "LITMUS_USERNAME, "
	}

	if len(LITMUS_PASSWORD) == 0 {
		msg = msg + "LITMUS_PASSWORD, "
	}
	if msg != "" {
		utils.Red.Println("❌ " + msg + " should be set as env var")
		os.Exit(1)
	}

	var authInput types.AuthInput
	authInput.Endpoint = LITMUS_FRONTEND_URL
	authInput.Username = LITMUS_USERNAME
	authInput.Password = LITMUS_PASSWORD

	resp, err := apis.Auth(authInput)
	utils.PrintError(err)
	// Decoding token
	token, _ := jwt.Parse(resp.AccessToken, nil)
	if token == nil {
		utils.Red.Println("\n❌ Cannot get token for user: " + authInput.Username + "\n")
		os.Exit(1)
	}

	var credentials types.Credentials
	credentials.Username = authInput.Username
	credentials.Endpoint = authInput.Endpoint
	credentials.Token = resp.AccessToken

	return credentials
}
