import os
from threading import Lock
import threading
import time
from flask import Flask, jsonify, request

from typing import List, Dict, Optional
from enum import Enum
from pydantic import BaseModel, ValidationError
import requests
import logging



# Set up logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Environment variables
MASTER_HOST = os.getenv("NOVELVMP_MASTER_HOST")
TEMPLATE_NAME = os.getenv("NOVELVMP_TEMPLATE_NAME")
SCANNER_NAME = os.getenv("NOVELVMP_SCANNER_NAME")
PORT = os.getenv("NOVELVMP_SCANNER_PORT")
MAX_REQUESTS = int(os.getenv("NOVELVMP_MAX_REQUESTS"))



# Enum for artifact types
class ArtifactType(Enum):
    DOMAIN = "domain"
    IP = "ip"
    HOST = "host"  # ip/domain:port
    URL = "url"
    CMS = "cms"
    HTTPMSG = "httpmsg"  # request/response pair
    SCREENSHOT = "screenshot"
    FINDING = "finding"
    TECHNOLOGY = "technology"

class Location(BaseModel):
    IP: Optional[str]
    URL: Optional[str]

# Enum for severity levels
class Severity(Enum):
    UNKNOWN = "unknown"
    INFO = "info"
    LOW = "low"
    MEDIUM = "medium"
    HIGH = "high"
    CRITICAL = "critical"

class Artifact(BaseModel):
    type: ArtifactType  # like domain
    value: Optional[str] = None  # like google.com
    location: Location = None  # url/ip of site
    scanner: str = None  # like nuclei
    severity: Optional[Severity] = None  # like high
    title: Optional[str] = None  # like "XSS..."
    description: Optional[str] = None  # optional description
    cve: Optional[str] = None  # optional CVE
    cvss: Optional[str] = None  # optional CVSS
    cvss_score: Optional[float] = None  # optional CVSS score
    data: Optional[bytes] = None  # optional data (e.g. http body)
    request: Optional[str] = None  # optional request
    response: Optional[str] = None  # optional response
    response_dom: Optional[str] = None  # optional response dom
    additional_data: Optional[Dict[str, object]] = None  # optional additional data
    version: Optional[str] = None  # optional version
    categories: Optional[List[str]] = None  # optional categories

    # NamedArtifact
    scanner_template: Optional[str] = None
    scanner_instance: Optional[str] = None

    
# class ArtifactNamed(BaseModel):
#     Artifact: Artifact
#     scanner_template: str
#     scanner_instance: str

class ScannerInstanceControlMsg(BaseModel):
    scanner_template: str
    scanner_instance: str
    scanner_msg: str

def check_environment():
    if not all([MASTER_HOST, TEMPLATE_NAME, SCANNER_NAME, PORT]):
        logger.critical("NOVELVMP_MASTER_HOST, NOVELVMP_TEMPLATE_NAME, NOVELVMP_SCANNER_NAME, and NOVELVMP_SCANNER_PORT environment variables must be set")
        raise EnvironmentError("Environment variables not set")

def send_artifact(artifact: Artifact):
    check_environment()

    logger.info("Sending Artifact to master: %s - %s", artifact.title, artifact.value)

    named_artifact = artifact.model_copy()
    named_artifact.scanner_template=TEMPLATE_NAME
    named_artifact.scanner_instance=SCANNER_NAME

    # send artifact to master
    try:
        artifact_json = named_artifact.model_dump_json()
        response = requests.post(f"http://{MASTER_HOST}/artifact", headers={"Content-Type": "application/json"}, data=artifact_json)
        response.raise_for_status()
    except requests.exceptions.RequestException as e:
        logger.critical("Failed to send artifact: %s", e)
        raise

registered = False
def send_msg_register():
    global registered
    check_environment()
    if registered:
        logger.critical("Already registered")
        raise RuntimeError("Already registered")
    registered = True
    send_ctrl_msg("register")

def send_msg_finish_task():
    check_environment()
    send_ctrl_msg("finish_task")

def send_ctrl_msg(msg: str):
    ctrl_msg = ScannerInstanceControlMsg(
        scanner_template=TEMPLATE_NAME,
        scanner_instance=SCANNER_NAME,
        scanner_msg=msg,
    )

    try:
        msg_json = ctrl_msg.model_dump_json()
        response = requests.post(f"http://{MASTER_HOST}/register", headers={"Content-Type": "application/json"}, data=msg_json)
        response.raise_for_status()
    except requests.exceptions.RequestException as e:
        logger.critical("Failed to send control message: %s", e)
        raise



# Flask app setup
app = Flask(__name__)

work_func_mutex = Lock()
work_func = None  # This will be set by NewServer

@app.route("/status", methods=["GET"])
def status():
    return "running", 200

@app.route("/artifact", methods=["POST"])
def post_artifact():
    try:
        data = request.json
        data['location'] = Location(**data['location'])
        artifact = Artifact(**data)
    except ValidationError as e:
        return jsonify(e.errors()), 400
    
    logger.info("Received artifact: %s - %s", artifact.title, artifact.value)
    
    def process_artifact():
        with work_func_mutex:
            work_func(artifact)
            logger.info("Finished Task")
            send_msg_finish_task()
    
    # Run the work function in a separate thread
    import threading
    threading.Thread(target=process_artifact).start()
    
    return "", 200

def run_server(work_function):
    global work_func
    work_func = work_function

    logger.info("MasterHost: %s", MASTER_HOST)
    logger.info("TemplateName: %s", TEMPLATE_NAME)
    logger.info("ScannerName: %s", SCANNER_NAME)
    logger.info("Port: %s", PORT)
    logger.info("MaxRequests: %d", MAX_REQUESTS)
    
    def register():
        time.sleep(1)
        send_msg_register()
    threading.Thread(target=register).start()
    
    app.run(port=PORT)