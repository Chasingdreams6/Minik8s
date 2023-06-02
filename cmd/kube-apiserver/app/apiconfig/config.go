package apiconfig

var Server_URL = "http://localhost:8080"

// API format : /api/{version}/{group}/{resource}

const PATH = "/api/v1"

const NODE_PATH = "/api/v1/nodes"
const POD_PATH = "/api/v1/pods"
const SERVICE_PATH = "/api/v1/services"
const REPLICASET_PATH = "/api/v1/replicasets"
const DEPLOYMENT_PATH = "/api/v1/deployments"
const JOB_PATH = "/api/v1/jobs"
const JOB_FILE_PATH = "/api/v1/jobfile"
const HPA_PATH = "/api/v1/hpa"
const DNS_PATH = "/api/v1/dns"
const WORKFLOW_PATH = "/api/v1/workflow"
const FUNCTION_PATH = "/api/v1/function"

// dir PATH
const JOB_FILE_DIR_PATH = "/home/job"
