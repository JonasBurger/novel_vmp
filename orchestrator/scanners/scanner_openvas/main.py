import time
from typing import Any, Tuple
import uuid
from gvm.connections import TLSConnection
from gvm.protocols.latest import Gmp
from gvm.transforms import EtreeTransform
from lxml import etree
from gvm.xml import pretty_print

from adapter import MAX_REQUESTS, Artifact, ArtifactType, Location, Severity, run_server, send_artifact, logger


SCANNER_NAME = 'scanner_openvas'


PORT_LIST_NAME = 'All TCP and Nmap top 100 UDP'
CONFIG_NAME = 'Discovery'
# CONFIG_NAME = 'Full and fast'
OPENVAS_SCANNER_NAME = 'OpenVAS Default'


def get_port_list_id(gmp: Gmp) -> str:
    # Get all port lists
    # https://docs.greenbone.net/API/GMP/gmp-22.5.html#command_get_port_lists
    response = gmp.get_port_lists()
    if response.attrib['status'] != '200':
        logger.error(f'response failed: {response.attrib["status_text"]}')
        assert False

    # Find the port list with the name 'All TCP and Nmap top 100 UDP'
    port_list_id = None
    for port_list in response.xpath('port_list'):
        if port_list.xpath('name/text()')[0] == PORT_LIST_NAME:
            port_list_id = port_list.xpath('@id')[0]
            break
    assert port_list_id is not None
    return port_list_id


def get_config_full_and_fast_id(gmp: Gmp) -> str:
    # Get all configs
    # https://docs.greenbone.net/API/GMP/gmp-22.5.html#command_get_configs
    response = gmp.get_scan_configs()
    assert response.attrib['status'] == '200'

    # Find the config with the name 'Full and fast'
    config_id = None
    for config in response.xpath('config'):
        if config.xpath('name/text()')[0] == CONFIG_NAME:
            config_id = config.xpath('@id')[0]
            break
    assert config_id is not None
    return config_id


def clone_config(gmp, config_id, new_name):
    response = gmp.clone_scan_config(config_id)
    cloned_config_id = response.xpath('//@id')[0]
    return cloned_config_id


def set_time_between_request(gmp, config_id, time_between_requests):
    response = gmp.modify_scan_config_set_scanner_preference(
        config_id, 'time_between_request', value=str(time_between_requests))
    assert response.attrib['status'] == '200'
    return response


def set_other_config_options(gmp, config_id):
    # got ids via response = gmp.get_scan_config_preferences(config_id=config_id, nvt_oid='1.3.6.1.4.1.25623.1.0.100315')
    # and pretty_print(response)

    # do i want to keep theese?
    response = gmp.modify_scan_config_set_nvt_preference(
        config_id, nvt_oid='1.3.6.1.4.1.25623.1.0.100315', name='1.3.6.1.4.1.25623.1.0.100315:1:checkbox:Do a TCP ping', value='yes')
    assert response.attrib['status'] == '200'

    response = gmp.modify_scan_config_set_nvt_preference(
        config_id, nvt_oid='1.3.6.1.4.1.25623.1.0.100315', name='1.3.6.1.4.1.25623.1.0.100315:2:checkbox:TCP ping tries also TCP-SYN ping', value='yes')
    assert response.attrib['status'] == '200'

    response = gmp.modify_scan_config_set_nvt_preference(
        config_id, nvt_oid='1.3.6.1.4.1.25623.1.0.100315', name='1.3.6.1.4.1.25623.1.0.100315:6:checkbox:Report about unrechable Hosts', value='yes')
    assert response.attrib['status'] == '200'

    response = gmp.modify_scan_config_set_nvt_preference(
        config_id, nvt_oid='1.3.6.1.4.1.25623.1.0.100315', name='1.3.6.1.4.1.25623.1.0.100315:9:checkbox:Report about reachable Hosts', value='yes')
    assert response.attrib['status'] == '200'

    response = gmp.modify_scan_config_set_nvt_preference(
        config_id, nvt_oid='1.3.6.1.4.1.25623.1.0.100315', name='1.3.6.1.4.1.25623.1.0.100315:13:checkbox:Log failed nmap calls', value='yes')
    assert response.attrib['status'] == '200'

    response = gmp.modify_scan_config_set_nvt_preference(
        config_id, nvt_oid='1.3.6.1.4.1.25623.1.0.100315', name='1.3.6.1.4.1.25623.1.0.100315:12:checkbox:Log nmap output', value='yes')
    assert response.attrib['status'] == '200'

    response = gmp.modify_scan_config_set_nvt_preference(
        config_id, nvt_oid='1.3.6.1.4.1.25623.1.0.100315', name='1.3.6.1.4.1.25623.1.0.100315:11:checkbox:nmap: try also with only -sP', value='yes')
    assert response.attrib['status'] == '200'


def get_scanner_id(gmp: Gmp) -> str:
    # Get all scanners
    # https://docs.greenbone.net/API/GMP/gmp-22.5.html#command_get_scanners
    response = gmp.get_scanners()
    assert response.attrib['status'] == '200'

    # Find the scanner with the name 'OpenVAS Default'
    scanner_id = None
    for scanner in response.xpath('scanner'):
        if scanner.xpath('name/text()')[0] == OPENVAS_SCANNER_NAME:
            scanner_id = scanner.xpath('@id')[0]
            break
    assert scanner_id is not None
    return scanner_id


def create_target(gmp: Gmp, name: str, host: str) -> str:
    # Create a new target
    # https://docs.greenbone.net/API/GMP/gmp-22.5.html#command_create_target
    response = gmp.create_target(name=name,
                                 hosts=[host],
                                 port_list_id=get_port_list_id(gmp),
                                 alive_test="TCP-SYN Service Ping"
                                 )
    assert response.attrib['status'] == '201'
    target_ids = response.xpath('//@id')
    assert len(target_ids) == 1
    return target_ids[0]


def create_task(gmp: Gmp, name: str, target_id: str, config_id: str, scanner_id: str) -> str:
    # Create a new task
    # https://docs.greenbone.net/API/GMP/gmp-22.5.html#command_create_task
    response = gmp.create_task(name, config_id, target_id, scanner_id)
    assert response.attrib['status'] == '201'
    task_ids = response.xpath('//@id')
    assert len(task_ids) == 1
    return task_ids[0]


def start_task(gmp: Gmp, task_id: str):
    # Start a task
    # https://docs.greenbone.net/API/GMP/gmp-22.5.html#command_start_task
    response = gmp.start_task(task_id)
    assert response.attrib['status'] == '202'


def get_task_status(gmp: Gmp, task_id: str) -> Tuple[str, str]:
    # Get the status of the task
    response = gmp.get_task(task_id)
    status = response.xpath('//task/status/text()')[0]
    progress = response.xpath('//task/progress/text()')[0]
    return status, progress


def get_report_id(gmp: Gmp, task_id: str) -> str:
    # Get the report ID associated with the task
    response = gmp.get_task(task_id)
    report_id = response.xpath('//task/last_report/report/@id')[0]
    return report_id


def get_report(gmp: Gmp, report_id: str) -> Any:
    # Get the report in XML format
    response = gmp.get_report(report_id, ignore_pagination=True)
    return response


def pretty_print_xml(xml_element):
    # Pretty print the XML element
    print(etree.tostring(xml_element, pretty_print=True).decode())


def threat_to_severity(threat):
    if threat == 'Log':
        return Severity.INFO
    elif threat == 'Low':
        return Severity.LOW
    elif threat == 'Medium':
        return Severity.MEDIUM
    elif threat == 'High':
        return Severity.HIGH
    elif threat == 'Critical':
        return Severity.CRITICAL
    else:
        return Severity.UNKNOWN


def parse_report(report_response, url):
    send_artifact_count = 0
    for result in report_response.findall(".//result"):
        if result.find("name") is None:
            print(f"name is None: {result}")
            continue

        send_artifact_count += 1
        name = result.find("name").text
        creation_time = result.find("creation_time").text
        host = result.find("host").text
        port = result.find("port").text
        cvss_score = result.find("severity").text
        nvt_tags = result.find("nvt/tags").text
        description = result.find("description").text
        threat = result.find("threat").text
        nvt_version = result.find("scan_nvt_version").text
        nvt_solution = result.find("nvt/solution").text
        references = result.find("nvt/refs")
        refs_list = []
        if references is not None:
            for reference in references.findall("ref"):
                id = reference.attrib['id']
                type = reference.attrib['type']
                refs_list.append((id, type))

        artifact = Artifact(
            type=ArtifactType.FINDING,
            value=port,
            location=Location(IP=host, URL=url),
            scanner=SCANNER_NAME,
            severity=threat_to_severity(threat),
            title=name,
            description=description,
            # cve: Optional[str] = None  # optional CVE
            # cvss: Optional[str] = None  # optional CVSS
            cvss_score=float(cvss_score),
            # data: Optional[bytes] = None  # optional data (e.g. http body)
            # request: Optional[str] = None  # optional request
            # response: Optional[str] = None  # optional response
            # response_dom: Optional[str] = None  # optional response dom
            additional_data={
                'creation_time': creation_time,
                'nvt_tags': nvt_tags,
                'nvt_version': nvt_version,
                'nvt_solution': nvt_solution,
                'references': refs_list,
            }
            # version: Optional[str] = None  # optional version
            # categories: Optional[List[str]] = None  # optional categories)
        )
        send_artifact(artifact)
    logger.info(f'Sent {send_artifact_count} artifacts')


def main():
    # connection = UnixSocketConnection()
    connection = TLSConnection()
    transform = EtreeTransform()

    with Gmp(connection, transform=transform) as gmp:
        # # Retrieve GMP version supported by the remote daemon
        version = gmp.get_version()
        # # Prints the XML in beautiful form
        pretty_print(version)

        # Login
        response = gmp.authenticate('admin', 'admin')
        if response.attrib['status'] != '200':
            logger.error(f'Login failed: {response.attrib["status_text"]}')
            assert False

        def work(artifact: Artifact):
            # todo: disable some scanners for application scanning
            assert artifact.type == ArtifactType.IP or artifact.type == ArtifactType.DOMAIN

            scan_uid = str(uuid.uuid4())
            target_id = create_target(gmp, 'Test_' + scan_uid, artifact.value)
            config_id = get_config_full_and_fast_id(gmp)
            cloned_config_id = clone_config(gmp, config_id, 'Test_' + scan_uid)
            timeout_milliseconds = int((1.0 / MAX_REQUESTS) * 1000)
            set_time_between_request(
                gmp, cloned_config_id, timeout_milliseconds)
            set_other_config_options(gmp, cloned_config_id)

            scanner_id = get_scanner_id(gmp)
            task_id = create_task(gmp, 'Test_' + scan_uid,
                                  target_id, cloned_config_id, scanner_id)
            start_task(gmp, task_id)

            # wait for the task to finish
            requested_count = 0
            while True:
                status, progress = get_task_status(gmp, task_id)
                logger.info(f'Task status: {status}, progress: {progress}%')
                if status in ['Done', 'Stopped', 'Interrupted']:
                    break
                if status == 'Requested':
                    requested_count += 1
                    if requested_count > 10:
                        requested_count = 0
                        logger.info(
                            'Task is stuck in requested state. Restarting task')
                        gmp.delete_task(task_id)
                        task_id = create_task(gmp, 'Test_' + scan_uid,
                                              target_id, cloned_config_id, scanner_id)
                        start_task(gmp, task_id)
                        continue
                time.sleep(30)

            # Retrieve the results
            report_id = get_report_id(gmp, task_id)
            # report_id = "ec846e8a-f265-477f-a716-b41f05c8e409"
            report_response = get_report(gmp, report_id)
            # pretty_print_xml(report_response)

            url = f"http://{artifact.value}/"
            parse_report(report_response, url)

            # Clean up
            gmp.delete_report(report_id)
            gmp.delete_task(task_id)
            gmp.delete_target(target_id)

        run_server(work)


if __name__ == '__main__':
    main()
