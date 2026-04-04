import csv
import datetime as dt
import io
import json
import subprocess
import sys
import time
import uuid
from pathlib import Path

import requests


BASE = "http://localhost:8090/api/v1/cyber"
TENANT = "0852d3f4-22cd-45c1-bac2-123196176aee"
TOKENS = {
    "analyst": Path("/tmp/clario360-perf/analyst.token").read_text().strip(),
    "manager": Path("/tmp/clario360-perf/manager.token").read_text().strip(),
    "admin": Path("/tmp/clario360-perf/admin.token").read_text().strip(),
}
SUMMARY_PATH = Path("/tmp/clario360-perf/live_cyber_suite_full_summary.json")
MANUAL_CONFIRM = "I have manually performed the remediation steps."


class QAError(Exception):
    pass


summary = {"started_at": dt.datetime.utcnow().isoformat() + "Z", "modules": {}, "artifacts": {}, "errors": []}
artifacts = {
    "asset_ids": [],
    "rule_ids": [],
}


def log(message):
    print(message, flush=True)


def module_summary(name):
    return summary["modules"].setdefault(name, {})


def now_iso(offset_seconds=0):
    return (dt.datetime.utcnow() + dt.timedelta(seconds=offset_seconds)).replace(microsecond=0).isoformat() + "Z"


def unique_suffix():
    return str(int(time.time()))


def require(cond, message, details=None):
    if not cond:
        raise QAError(f"{message}: {details}" if details is not None else message)


def data_of(body):
    if isinstance(body, dict) and "data" in body:
        return body["data"]
    return body


def request(method, path, token_name, expected, *, params=None, json_body=None, files=None, timeout=120):
    token = TOKENS[token_name]
    headers = {"Authorization": f"Bearer {token}", "Accept": "application/json"}
    kwargs = {"params": params, "timeout": timeout, "headers": headers}
    if files is not None:
        kwargs["files"] = files
    elif json_body is not None:
        kwargs["json"] = json_body
    response = requests.request(method, BASE + path, **kwargs)
    parsed = None
    if response.text:
        try:
            parsed = response.json()
        except Exception:
            parsed = {"raw": response.text}
    if response.status_code not in set(expected):
        raise QAError(
            f"{method} {path} returned {response.status_code}, expected {sorted(set(expected))}: {json.dumps(parsed, default=str)}"
        )
    return response.status_code, parsed


def db_query(sql):
    proc = subprocess.run(
        [
            "docker",
            "exec",
            "-i",
            "clario360-postgres",
            "psql",
            "-U",
            "clario",
            "-d",
            "cyber_db",
            "-Atc",
            sql,
        ],
        capture_output=True,
        text=True,
    )
    if proc.returncode != 0:
        raise QAError(f"db query failed: {proc.stderr.strip()}")
    lines = [line.strip() for line in proc.stdout.splitlines() if line.strip()]
    return lines


def produce_security_batch(events_batch):
    cloud_event = {
        "id": str(uuid.uuid4()),
        "source": "clario360/qa-harness",
        "specversion": "1.0",
        "type": "com.clario360.cyber.security.events.received",
        "datacontenttype": "application/json",
        "time": now_iso(),
        "timestamp": now_iso(),
        "tenantid": TENANT,
        "userid": "33333333-3333-4333-8333-333333333333",
        "correlationid": str(uuid.uuid4()),
        "data": events_batch,
    }
    proc = subprocess.run(
        ["docker", "exec", "-i", "clario360-redpanda", "rpk", "topic", "produce", "cyber.security_events"],
        input=(json.dumps(cloud_event) + "\n").encode(),
        capture_output=True,
    )
    if proc.returncode != 0:
        raise QAError(f"security event produce failed: {proc.stderr.decode().strip()}")
    return proc.stdout.decode().strip()


def poll(description, fn, *, timeout=180, interval=2):
    deadline = time.time() + timeout
    last = None
    while time.time() < deadline:
        done, value = fn()
        last = value
        if done:
            return value
        time.sleep(interval)
    raise QAError(f"timeout waiting for {description}: {last}")


def first_item(list_response):
    if isinstance(list_response, dict) and "data" in list_response and isinstance(list_response["data"], list):
        return list_response["data"][0] if list_response["data"] else None
    if isinstance(list_response, list):
        return list_response[0] if list_response else None
    return None


def find_alert_for_asset(alert_list_data, asset_id):
    items = alert_list_data.get("data", [])
    for item in items:
        if item.get("asset_id") == asset_id:
            return item
    return None


def make_event(*, asset_id, source_ip, event_type, source="qa-security", severity="high", dest_ip=None, dest_port=None, protocol="tcp", username=None, raw=None, ts_offset=0):
    payload = {
        "id": str(uuid.uuid4()),
        "tenant_id": TENANT,
        "timestamp": now_iso(ts_offset),
        "source": source,
        "type": event_type,
        "severity": severity,
        "asset_id": asset_id,
        "raw_event": raw or {},
    }
    if source_ip is not None:
        payload["source_ip"] = source_ip
    if dest_ip is not None:
        payload["dest_ip"] = dest_ip
    if dest_port is not None:
        payload["dest_port"] = dest_port
    if protocol is not None:
        payload["protocol"] = protocol
    if username is not None:
        payload["username"] = username
    return payload


def load_fixtures():
    users = db_query("select id from users order by created_at asc limit 2;")
    require(len(users) >= 2, "expected at least two users for alert assign/escalate")
    db_assets = db_query(
        f"select id from assets where tenant_id='{TENANT}' and type='database' and deleted_at is null order by created_at asc limit 1;"
    )
    require(db_assets, "expected at least one existing database asset")
    threats = db_query(
        f"select id from threats where tenant_id='{TENANT}' and deleted_at is null order by created_at desc limit 1;"
    )
    require(threats, "expected at least one existing threat")
    return {"user_ids": users, "existing_db_asset_id": db_assets[0], "existing_threat_id": threats[0]}


def run_assets(fixtures, suffix):
    log("assets: create individual assets")
    mod = module_summary("assets")
    ip_base = f"10.254.200"
    db_payload = {
        "name": f"qa-db-{suffix}",
        "type": "database",
        "ip_address": f"{ip_base}.11",
        "hostname": f"qa-db-{suffix}.local",
        "os": "linux",
        "os_version": "Ubuntu 22.04",
        "department": "security",
        "location": "qa-lab",
        "criticality": "low",
        "tags": ["qa", "database", "production"],
        "metadata": {
            "open_ports": [5432],
            "database_type": "postgres",
            "encryption": True,
            "ssl": True,
            "rbac": True,
            "backup_configured": True,
            "query_logging": True,
            "schema": {"tables": ["customers", "orders"], "columns": ["email", "ssn"]},
        },
    }
    _, body = request("POST", "/assets", "admin", {201}, json_body=db_payload)
    db_asset = data_of(body)
    artifacts["asset_ids"].append(db_asset["id"])

    _, body = request("GET", f"/assets/{db_asset['id']}", "analyst", {200})
    db_asset = data_of(body)
    mod["db_asset"] = {"id": db_asset["id"], "criticality": db_asset["criticality"]}
    require(db_asset["criticality"] == "critical", "database asset should auto-classify to critical", db_asset["criticality"])

    web_payload = {
        "name": f"qa-web-{suffix}",
        "type": "server",
        "ip_address": f"{ip_base}.12",
        "hostname": f"qa-web-{suffix}.local",
        "os": "linux",
        "os_version": "Ubuntu 22.04",
        "owner": "qa-owner",
        "department": "security",
        "location": "qa-lab",
        "criticality": "medium",
        "tags": ["qa", "production", "internet-facing"],
        "metadata": {
            "open_ports": [22, 80, 443],
            "protocols": ["HTTP", "TLSv1.0"],
            "default_credentials": False,
            "query_logging": True,
        },
    }
    _, body = request("POST", "/assets", "admin", {201}, json_body=web_payload)
    web_asset = data_of(body)
    artifacts["asset_ids"].append(web_asset["id"])
    mod["web_asset"] = {"id": web_asset["id"], "criticality": web_asset["criticality"]}

    _, body = request("GET", f"/assets/{web_asset['id']}", "analyst", {200})
    fetched_web = data_of(body)
    require(fetched_web["name"] == web_payload["name"], "asset GET should return created web asset")

    update_payload = {"name": f"qa-web-{suffix}-updated", "owner": "qa-owner-2", "location": "qa-dmz"}
    _, body = request("PUT", f"/assets/{web_asset['id']}", "admin", {200}, json_body=update_payload)
    updated_web = data_of(body)
    require(updated_web["name"] == update_payload["name"], "asset update should change name")

    _, body = request("PATCH", f"/assets/{web_asset['id']}/tags", "admin", {200}, json_body={"add": ["dmz"], "remove": []})
    patched_web = data_of(body)
    require("dmz" in patched_web["tags"], "asset patch tags should add dmz")

    _, body = request("GET", "/assets", "analyst", {200}, params={"search": f"qa-web-{suffix}-updated", "type": "server", "tag": "qa", "per_page": 10})
    listed = body
    require(listed["total"] >= 1, "asset list search/filter should find qa web asset")
    mod["list_search_total"] = listed["total"]

    _, body = request(
        "POST",
        f"/assets/{web_asset['id']}/relationships",
        "admin",
        {201},
        json_body={"target_asset_id": db_asset["id"], "relationship_type": "depends_on", "metadata": {"purpose": "qa_e2e"}},
    )
    relationship = data_of(body)
    mod["relationship_id"] = relationship["id"]

    _, body = request("GET", f"/assets/{web_asset['id']}/relationships", "analyst", {200})
    rels = data_of(body)
    require(len(rels.get("outgoing", [])) >= 1, "relationship list should include outgoing dependency")

    bulk_json = [
        {
            "name": f"qa-endpoint-{suffix}",
            "type": "endpoint",
            "ip_address": f"{ip_base}.21",
            "hostname": f"qa-endpoint-{suffix}.local",
            "os": "windows",
            "os_version": "Windows 11",
            "owner": "qa-endpoint-owner",
            "department": "it",
            "location": "qa-lab",
            "criticality": "low",
            "tags": ["qa", "bulkjson"],
            "metadata": {"open_ports": [3389]},
        },
        {
            "name": f"qa-container-{suffix}",
            "type": "container",
            "ip_address": f"{ip_base}.22",
            "hostname": f"qa-container-{suffix}.local",
            "os": "linux",
            "os_version": "Alpine 3.19",
            "department": "engineering",
            "location": "qa-cluster",
            "criticality": "low",
            "tags": ["qa", "bulkjson"],
            "metadata": {"open_ports": [8080]},
        },
    ]
    _, body = request("POST", "/assets/bulk", "admin", {201}, json_body=bulk_json)
    bulk_json_result = data_of(body)
    json_asset_ids = bulk_json_result["ids"]
    artifacts["asset_ids"].extend(json_asset_ids)
    mod["bulk_json_count"] = bulk_json_result["count"]
    require(bulk_json_result["count"] == 2, "bulk JSON should create two assets")

    csv_buffer = io.StringIO()
    writer = csv.writer(csv_buffer)
    writer.writerow(["name", "type", "criticality", "ip_address", "hostname", "os", "owner", "department", "location"])
    writer.writerow([f"qa-csv-app-{suffix}", "application", "medium", f"{ip_base}.31", f"qa-csv-app-{suffix}.local", "linux", "qa-app-owner", "engineering", "qa-zone"])
    writer.writerow([f"qa-csv-iot-{suffix}", "iot_device", "low", f"{ip_base}.32", f"qa-csv-iot-{suffix}.local", "linux", "qa-iot-owner", "operations", "qa-zone"])
    _, body = request(
        "POST",
        "/assets/bulk",
        "admin",
        {201},
        files={"file": ("qa_assets.csv", csv_buffer.getvalue().encode(), "text/csv")},
    )
    bulk_csv_result = data_of(body)
    csv_asset_ids = bulk_csv_result["ids"]
    artifacts["asset_ids"].extend(csv_asset_ids)
    mod["bulk_csv_count"] = bulk_csv_result["count"]
    require(bulk_csv_result["count"] == 2, "bulk CSV should create two assets")

    _, _ = request("PUT", "/assets/bulk/tags", "admin", {204}, json_body={"asset_ids": json_asset_ids, "add": ["bulk-qa"], "remove": []})
    _, body = request("GET", "/assets", "analyst", {200}, params={"tag": "bulk-qa", "per_page": 10})
    require(body["total"] >= 2, "bulk tag update should make JSON assets queryable by bulk-qa tag")
    mod["bulk_tag_total"] = body["total"]

    _, _ = request("DELETE", "/assets/bulk", "admin", {204}, json_body={"asset_ids": csv_asset_ids})
    _, body = request("GET", "/assets", "analyst", {200}, params={"search": f"qa-csv-app-{suffix}", "per_page": 10})
    require(body["total"] == 0, "bulk deleted CSV asset should not appear in list")
    mod["bulk_delete_confirmed"] = True

    _, body = request("GET", "/assets/stats", "analyst", {200})
    stats = data_of(body)
    require(stats.get("total_assets", 0) >= 1, "asset stats should return totals")
    mod["stats_total"] = stats.get("total_assets")

    _, body = request("GET", "/assets/count", "analyst", {200}, params={"type": "database"})
    mod["count_database"] = data_of(body)["count"]

    log("assets: run successful network scan")
    _, body = request(
        "POST",
        "/assets/scan",
        "admin",
        {202},
        json_body={"scan_type": "network", "targets": ["127.0.0.1/32"], "ports": [8090], "options": {"allow_public_scan": True}},
    )
    scan_id = data_of(body)["scan_id"]

    def scan_done():
        _, scan_body = request("GET", f"/assets/scans/{scan_id}", "analyst", {200})
        scan = data_of(scan_body)
        return scan["status"] in {"completed", "failed", "cancelled"}, scan

    scan = poll("network scan completion", scan_done, timeout=120, interval=2)
    require(scan["status"] == "completed", "network scan should complete successfully", scan)
    mod["network_scan"] = {"id": scan_id, "status": scan["status"], "assets_discovered": scan.get("assets_discovered")}

    log("assets: run cancellable network scan")
    _, body = request(
        "POST",
        "/assets/scan",
        "admin",
        {202},
        json_body={
            "scan_type": "network",
            "targets": ["10.255.255.0/24"],
            "ports": [65000],
            "options": {},
        },
    )
    cancel_scan_id = data_of(body)["scan_id"]
    _, _ = request("POST", f"/assets/scans/{cancel_scan_id}/cancel", "admin", {204})

    def cancel_done():
        _, scan_body = request("GET", f"/assets/scans/{cancel_scan_id}", "analyst", {200})
        scan = data_of(scan_body)
        return scan["status"] in {"cancelled", "completed", "failed"}, scan

    cancelled = poll("network scan cancellation", cancel_done, timeout=120, interval=2)
    require(cancelled["status"] == "cancelled", "scan cancellation should set cancelled status", cancelled)
    mod["cancel_scan"] = {"id": cancel_scan_id, "status": cancelled["status"]}

    return {
        "db_asset_id": db_asset["id"],
        "web_asset_id": web_asset["id"],
        "bulk_json_asset_ids": json_asset_ids,
        "relationship_id": relationship["id"],
    }


def run_vulnerabilities(fixtures, suffix):
    log("vulnerabilities: create/list/update/detail/stats")
    mod = module_summary("vulnerabilities")
    existing_db_asset_id = fixtures["existing_db_asset_id"]
    cve_id = f"CVE-2026-{suffix[-4:]}"
    create_payload = {
        "cve_id": cve_id,
        "title": f"QA Manual Vulnerability {suffix}",
        "description": "Live QA manual vulnerability",
        "severity": "high",
        "cvss_score": 9.1,
        "cvss_vector": "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H",
        "source": "manual",
        "remediation": "Apply vendor hotfix",
        "proof": "QA synthetic evidence",
    }
    _, body = request("POST", f"/assets/{existing_db_asset_id}/vulnerabilities", "admin", {201}, json_body=create_payload)
    vuln = data_of(body)
    vuln_id = vuln["id"]
    mod["created_vulnerability_id"] = vuln_id

    _, body = request("GET", f"/assets/{existing_db_asset_id}/vulnerabilities", "analyst", {200}, params={"severity": "high", "per_page": 10})
    require(body["total"] >= 1, "per-asset vulnerability list should return created vulnerability")
    mod["asset_vuln_total"] = body["total"]

    _, body = request("PUT", f"/assets/{existing_db_asset_id}/vulnerabilities/{vuln_id}", "admin", {200}, json_body={"status": "in_progress"})
    require(data_of(body)["status"] == "in_progress", "per-asset vulnerability update should set in_progress")

    _, body = request(
        "GET",
        "/vulnerabilities",
        "analyst",
        {200},
        params={"cve_id": cve_id, "severity": "high", "asset_type": "database", "min_cvss": "7.0", "per_page": 10},
    )
    global_list = data_of(body)
    require(global_list["total"] >= 1, "global vulnerability list should find created vulnerability")
    mod["global_list_total"] = global_list["total"]

    _, body = request("GET", f"/vulnerabilities/{vuln_id}", "analyst", {200})
    detail = data_of(body)
    require(detail["id"] == vuln_id, "vulnerability detail should match created vulnerability")

    _, body = request("PUT", f"/vulnerabilities/{vuln_id}/status", "admin", {200}, json_body={"status": "resolved", "notes": "QA resolved"})
    require(data_of(body)["status"] == "resolved", "global vulnerability status update should resolve vulnerability")

    _, body = request("GET", "/vulnerabilities/stats", "analyst", {200})
    mod["stats"] = data_of(body)

    _, body = request("GET", "/vulnerabilities/aging", "analyst", {200})
    aging = data_of(body)
    require(aging["total_open"] >= 0, "vulnerability aging should return totals")
    mod["aging_total_open"] = aging["total_open"]

    _, body = request("GET", "/vulnerabilities/top-cves", "analyst", {200}, params={"limit": 10})
    mod["top_cves_count"] = len(data_of(body))


def run_threats_and_rules(fixtures, assets_ctx, suffix):
    log("threats: add indicators and bulk import")
    threats_mod = module_summary("threats")
    rules_mod = module_summary("rules")
    alerts_mod = module_summary("alerts")

    threat_id = fixtures["existing_threat_id"]
    malicious_ip = f"198.51.100.{int(suffix[-2:])%200 + 20}"
    stix_domain = f"qa-bad-{suffix}.example"

    _, body = request("GET", "/threats/stats", "analyst", {200})
    threats_mod["stats"] = data_of(body)
    _, body = request("GET", "/threats", "analyst", {200}, params={"per_page": 5})
    threats_mod["list_total"] = body["total"]
    _, body = request("GET", f"/threats/{threat_id}", "analyst", {200})
    threats_mod["threat"] = {"id": data_of(body)["id"], "status": data_of(body)["status"]}

    indicator_payload = {
        "type": "ip",
        "value": malicious_ip,
        "description": "QA malicious IP",
        "severity": "critical",
        "source": "manual",
        "confidence": 0.95,
        "tags": ["qa", "live-e2e"],
        "metadata": {"qa": True},
    }
    _, body = request("POST", f"/threats/{threat_id}/indicators", "admin", {201}, json_body=indicator_payload)
    indicator = data_of(body)
    threats_mod["manual_indicator_id"] = indicator["id"]

    _, body = request("GET", f"/threats/{threat_id}/indicators", "analyst", {200})
    threat_indicators = data_of(body)
    require(any(item["value"] == malicious_ip for item in threat_indicators), "manual threat indicator should be listed")

    _, body = request("POST", "/indicators/check", "analyst", {200}, json_body={"values": [malicious_ip]})
    checks = data_of(body)
    require(checks and checks[0]["indicators"], "indicator check should match malicious IP")
    threats_mod["indicator_check_matches"] = len(checks[0]["indicators"])

    stix_bundle = {
        "type": "bundle",
        "objects": [
            {"type": "campaign", "id": f"campaign--{uuid.uuid4()}", "name": f"QA Campaign {suffix}", "description": "QA campaign"},
            {
                "type": "indicator",
                "id": f"indicator--{uuid.uuid4()}",
                "name": f"QA Domain {suffix}",
                "description": "QA STIX domain",
                "pattern": f"[domain-name:value = '{stix_domain}']",
                "labels": ["high"],
                "confidence": 85,
            },
        ],
    }
    _, body = request("POST", "/indicators/bulk", "admin", {201}, json_body={"payload": stix_bundle, "source": "stix_feed"})
    threats_mod["bulk_import_count"] = data_of(body)["count"]

    _, body = request("GET", "/indicators", "analyst", {200}, params={"search": stix_domain, "per_page": 10})
    indicator_list = body
    require(indicator_list["total"] >= 1, "bulk imported STIX indicator should be listable")
    threats_mod["list_indicators_total"] = indicator_list["total"]

    _, body = request("PUT", f"/threats/{threat_id}/status", "admin", {200}, json_body={"status": "monitoring"})
    require(data_of(body)["status"] == "monitoring", "threat status should update to monitoring")
    _, body = request("PUT", f"/threats/{threat_id}/status", "admin", {200}, json_body={"status": "active"})
    require(data_of(body)["status"] == "active", "threat status should revert to active")

    log("rules: create threshold, sigma, correlation rules")
    _, body = request("GET", "/rules/templates", "analyst", {200})
    templates = data_of(body)
    require(len(templates) >= 1, "rule templates should be available")
    rules_mod["template_count"] = len(templates)

    rule1_payload = {
        "name": f"QA Threshold {suffix}",
        "description": "QA threshold rule",
        "rule_type": "threshold",
        "severity": "high",
        "enabled": True,
        "rule_content": {
            "field": "source_ip",
            "condition": {"type": "login_failed", "source": "qa-security"},
            "threshold": 5,
            "window": "5m",
            "metric": "count",
        },
        "mitre_tactic_ids": ["TA0006"],
        "mitre_technique_ids": ["T1110"],
        "base_confidence": 0.82,
        "tags": ["qa", "threshold"],
    }
    _, body = request("POST", "/rules", "admin", {201}, json_body=rule1_payload)
    rule1 = data_of(body)
    artifacts["rule_ids"].append(rule1["id"])

    rule2_payload = {
        "name": f"QA Sigma {suffix}",
        "description": "QA sigma rule",
        "rule_type": "sigma",
        "severity": "medium",
        "enabled": True,
        "rule_content": {"detection": {"selection": {"type": "qa_sigma"}, "condition": "selection"}},
        "mitre_tactic_ids": ["TA0003"],
        "mitre_technique_ids": ["T1059"],
        "base_confidence": 0.75,
        "tags": ["qa", "sigma"],
    }
    _, body = request("POST", "/rules", "admin", {201}, json_body=rule2_payload)
    rule2 = data_of(body)
    artifacts["rule_ids"].append(rule2["id"])

    rule3_payload = {
        "name": f"QA Correlation {suffix}",
        "description": "QA correlation rule",
        "rule_type": "correlation",
        "severity": "critical",
        "enabled": True,
        "rule_content": {
            "events": [
                {"name": "failed", "condition": {"type": "login_failed"}},
                {"name": "success", "condition": {"type": "login_success"}},
                {"name": "privesc", "condition": {"type": "privilege_escalation"}},
            ],
            "sequence": ["failed", "success", "privesc"],
            "group_by": "username",
            "window": "30m",
            "min_failed_count": 2,
        },
        "mitre_tactic_ids": ["TA0006", "TA0004"],
        "mitre_technique_ids": ["T1110", "T1068"],
        "base_confidence": 0.9,
        "tags": ["qa", "correlation"],
    }
    _, body = request("POST", "/rules", "admin", {201}, json_body=rule3_payload)
    rule3 = data_of(body)
    artifacts["rule_ids"].append(rule3["id"])

    _, body = request("GET", f"/rules/{rule1['id']}", "analyst", {200})
    require(data_of(body)["id"] == rule1["id"], "rule GET should return created threshold rule")

    _, body = request("PUT", f"/rules/{rule2['id']}", "admin", {200}, json_body={"description": "QA sigma rule updated", "severity": "high"})
    require(data_of(body)["severity"] == "high", "rule update should change sigma severity")
    _, _ = request("PUT", f"/rules/{rule2['id']}/toggle", "admin", {200}, json_body={"enabled": False})
    _, body = request("PUT", f"/rules/{rule2['id']}/toggle", "admin", {200}, json_body={"enabled": True})
    require(data_of(body)["enabled"] is True, "rule toggle should re-enable sigma rule")

    _, body = request("GET", "/rules", "analyst", {200}, params={"search": f"QA {suffix}", "per_page": 10})
    rules_mod["created_rule_total"] = body["total"]

    log("rules/detection: waiting 65s for background rule + indicator reload")
    time.sleep(65)

    web_asset_id = assets_ctx["web_asset_id"]
    feedback_asset_id = assets_ctx["bulk_json_asset_ids"][0]
    threshold_source = "203.0.113.50"
    feedback_source = "203.0.113.60"
    corr_user = f"qa-corr-{suffix}"
    initial_batch = []
    for i, port in enumerate([2201, 2202, 2203, 2204, 2205]):
        initial_batch.append(
            make_event(
                asset_id=web_asset_id,
                source_ip=threshold_source,
                event_type="login_failed",
                dest_ip="10.1.1.10",
                dest_port=port,
                username=f"qa-brute-{suffix}",
                ts_offset=i,
            )
        )
    initial_batch.append(make_event(asset_id=web_asset_id, source_ip="203.0.113.80", event_type="qa_sigma", dest_ip="10.1.1.20", dest_port=8443, ts_offset=10))
    initial_batch.extend(
        [
            make_event(asset_id=web_asset_id, source_ip="203.0.113.81", event_type="login_failed", username=corr_user, ts_offset=20),
            make_event(asset_id=web_asset_id, source_ip="203.0.113.81", event_type="login_failed", username=corr_user, ts_offset=21),
            make_event(asset_id=web_asset_id, source_ip="203.0.113.81", event_type="login_success", username=corr_user, ts_offset=22),
            make_event(asset_id=web_asset_id, source_ip="203.0.113.81", event_type="privilege_escalation", username=corr_user, ts_offset=23),
        ]
    )
    initial_batch.append(make_event(asset_id=web_asset_id, source_ip=malicious_ip, event_type="network_connection", dest_ip="10.1.1.30", dest_port=443, ts_offset=30))
    alerts_mod["produce_initial"] = produce_security_batch(initial_batch)

    def wait_rule_alert(rule_id):
        def inner():
            _, resp_body = request("GET", "/alerts", "analyst", {200}, params={"rule_id": rule_id, "per_page": 20})
            data = resp_body
            return data["total"] >= 1, data
        return poll(f"alert for rule {rule_id}", inner, timeout=120, interval=2)

    rule1_alerts = wait_rule_alert(rule1["id"])
    rule2_alerts = wait_rule_alert(rule2["id"])
    rule3_alerts = wait_rule_alert(rule3["id"])
    primary_alert = find_alert_for_asset(rule1_alerts, web_asset_id)
    sigma_alert = find_alert_for_asset(rule2_alerts, web_asset_id)
    corr_alert = find_alert_for_asset(rule3_alerts, web_asset_id)
    require(primary_alert is not None, "threshold alert should exist for web asset")
    require(sigma_alert is not None, "sigma alert should exist for web asset")
    require(corr_alert is not None, "correlation alert should exist for web asset")

    def indicator_alert_ready():
        _, resp_body = request("GET", "/alerts", "analyst", {200}, params={"search": malicious_ip, "per_page": 20})
        data = resp_body
        return data["total"] >= 1, data

    indicator_alerts = poll("indicator alert", indicator_alert_ready, timeout=120, interval=2)
    alerts_mod["indicator_alert_total"] = indicator_alerts["total"]

    pre_dedup_count = primary_alert["event_count"]
    dedup_batch = [
        make_event(
            asset_id=web_asset_id,
            source_ip=threshold_source,
            event_type="login_failed",
            dest_ip="10.1.1.10",
            dest_port=port,
            username=f"qa-brute-{suffix}",
            ts_offset=60 + i,
        )
        for i, port in enumerate([2201, 2202, 2203, 2204, 2205])
    ]
    alerts_mod["produce_dedup"] = produce_security_batch(dedup_batch)

    def dedup_done():
        _, resp_body = request("GET", "/alerts", "analyst", {200}, params={"rule_id": rule1["id"], "per_page": 20})
        data = resp_body
        alert = find_alert_for_asset(data, web_asset_id)
        return alert is not None and alert["id"] == primary_alert["id"] and alert["event_count"] > pre_dedup_count, data

    dedup_after = poll("threshold dedup update", dedup_done, timeout=120, interval=2)
    updated_primary_alert = find_alert_for_asset(dedup_after, web_asset_id)
    require(dedup_after["total"] == 1, "threshold dedup should retain a single open alert for same rule+asset", dedup_after["total"])
    alerts_mod["dedup_event_count"] = updated_primary_alert["event_count"]

    feedback_batch = [
        make_event(
            asset_id=feedback_asset_id,
            source_ip=feedback_source,
            event_type="login_failed",
            dest_ip="10.2.2.2",
            dest_port=port,
            username=f"qa-feedback-{suffix}",
            ts_offset=90 + i,
        )
        for i, port in enumerate([3301, 3302, 3303, 3304, 3305])
    ]
    alerts_mod["produce_feedback"] = produce_security_batch(feedback_batch)

    def feedback_alert_ready():
        _, resp_body = request("GET", "/alerts", "analyst", {200}, params={"rule_id": rule1["id"], "per_page": 20})
        data = resp_body
        return find_alert_for_asset(data, feedback_asset_id) is not None, data

    feedback_alerts = poll("feedback alert", feedback_alert_ready, timeout=120, interval=2)
    feedback_alert = find_alert_for_asset(feedback_alerts, feedback_asset_id)
    require(feedback_alert is not None, "feedback alert should exist for second asset")

    _, body = request("POST", f"/rules/{rule3['id']}/test", "admin", {200}, json_body={"limit": 200})
    rule_test = data_of(body)
    require(rule_test["count"] >= 1, "correlation rule test should find at least one match", rule_test["count"])
    rules_mod["rule3_test_count"] = rule_test["count"]

    _, body = request("GET", "/alerts", "analyst", {200}, params={"rule_id": rule1["id"], "per_page": 20})
    alerts_mod["list_total_for_rule1"] = body["total"]

    _, body = request("GET", f"/alerts/{primary_alert['id']}", "analyst", {200})
    primary_detail = data_of(body)
    require(primary_detail["id"] == primary_alert["id"], "alert detail should return primary alert")

    user1, user2 = fixtures["user_ids"]
    _, body = request("PUT", f"/alerts/{primary_alert['id']}/assign", "admin", {200}, json_body={"assigned_to": user1})
    require(data_of(body)["assigned_to"] == user1, "alert assign should set assigned user")
    _, body = request("POST", f"/alerts/{primary_alert['id']}/escalate", "admin", {200}, json_body={"escalated_to": user2, "reason": "QA escalation"})
    require(data_of(body)["assigned_to"] == user2, "alert escalate should reassign alert")
    _, body = request("POST", f"/alerts/{primary_alert['id']}/comment", "admin", {201}, json_body={"content": "QA investigation note"})
    alerts_mod["comment_id"] = data_of(body)["id"]

    _, body = request("GET", f"/alerts/{primary_alert['id']}/comments", "analyst", {200})
    require(len(data_of(body)) >= 1, "alert comments should list QA note")
    _, body = request("GET", f"/alerts/{primary_alert['id']}/timeline", "analyst", {200})
    require(len(data_of(body)) >= 1, "alert timeline should have entries")

    _, body = request("POST", f"/alerts/{primary_alert['id']}/merge", "admin", {200}, json_body={"merge_ids": [sigma_alert["id"]]})
    merged = data_of(body)
    require(merged["id"] == primary_alert["id"], "alert merge should preserve primary alert")

    _, body = request("GET", f"/alerts/{primary_alert['id']}/related", "analyst", {200})
    alerts_mod["related_count"] = len(data_of(body))

    _, body = request("PUT", f"/alerts/{primary_alert['id']}/status", "admin", {200}, json_body={"status": "investigating", "notes": "QA status update"})
    require(data_of(body)["status"] == "investigating", "alert status should update to investigating")

    _, body = request("POST", f"/rules/{rule1['id']}/feedback", "admin", {200}, json_body={"alert_id": feedback_alert["id"], "feedback": "false_positive"})
    feedback_rule = data_of(body)
    require(feedback_rule["id"] == rule1["id"], "rule feedback should return the updated rule")
    require(feedback_rule["false_positive_count"] >= 1, "rule feedback should increment false positive count")
    rules_mod["feedback_false_positive_count"] = feedback_rule["false_positive_count"]

    _, body = request("GET", "/alerts/stats", "analyst", {200})
    alerts_mod["stats"] = data_of(body)
    _, body = request("GET", "/alerts/count", "analyst", {200}, params={"severity": "critical"})
    alerts_mod["critical_count"] = data_of(body)["count"]

    return {
        "primary_alert_id": primary_alert["id"],
        "feedback_alert_id": feedback_alert["id"],
        "rule_ids": [rule1["id"], rule2["id"], rule3["id"]],
        "malicious_ip": malicious_ip,
        "stix_domain": stix_domain,
    }


def run_mitre():
    log("mitre: tactics/techniques/coverage")
    mod = module_summary("mitre")
    _, body = request("GET", "/mitre/tactics", "analyst", {200})
    tactics = data_of(body)
    require(len(tactics) >= 1, "MITRE tactics should be available")
    mod["tactic_count"] = len(tactics)

    _, body = request("GET", "/mitre/techniques", "analyst", {200}, params={"per_page": 20})
    techniques = data_of(body)
    first = first_item(techniques)
    require(first is not None, "MITRE techniques should be available")
    mod["technique_total"] = techniques["total"] if isinstance(techniques, dict) else len(techniques)

    _, body = request("GET", f"/mitre/techniques/{first['id']}", "analyst", {200})
    detail = data_of(body)
    require(detail["id"] == first["id"], "MITRE technique detail should match list item")

    _, body = request("GET", "/mitre/coverage", "analyst", {200})
    coverage = data_of(body)
    mod["coverage"] = coverage


def run_ctem(assets_ctx, suffix):
    log("ctem: create/start/poll/report/compare")
    mod = module_summary("ctem")
    _, body = request("GET", "/ctem/assessments", "analyst", {200}, params={"per_page": 20})
    existing = data_of(body)
    existing_completed = [item for item in existing["data"] if item["status"] == "completed"]
    other_assessment_id = existing_completed[0]["id"] if existing_completed else None

    create_payload = {
        "name": f"QA CTEM {suffix}",
        "description": "QA CTEM assessment",
        "scope": {"asset_ids": [assets_ctx["web_asset_id"], assets_ctx["db_asset_id"]]},
        "scheduled": False,
        "tags": ["qa", "live-e2e"],
        "start": False,
    }
    _, body = request("POST", "/ctem/assessments", "admin", {201}, json_body=create_payload)
    assessment = data_of(body)
    assessment_id = assessment["id"]
    mod["assessment_id"] = assessment_id

    _, body = request("PUT", f"/ctem/assessments/{assessment_id}", "admin", {200}, json_body={"description": "QA CTEM assessment updated"})
    require(data_of(body)["description"] == "QA CTEM assessment updated", "CTEM update should change description")

    _, _ = request("POST", f"/ctem/assessments/{assessment_id}/start", "admin", {202}, json_body={})

    def assessment_done():
        _, resp_body = request("GET", f"/ctem/assessments/{assessment_id}", "analyst", {200})
        current = data_of(resp_body)
        return current["status"] in {"completed", "failed", "cancelled"}, current

    completed = poll("CTEM assessment completion", assessment_done, timeout=240, interval=3)
    require(completed["status"] == "completed", "CTEM assessment should complete", completed)
    mod["completed_status"] = completed["status"]
    mod["exposure_score"] = completed.get("exposure_score")

    for label, path in [
        ("scope", f"/ctem/assessments/{assessment_id}/scope"),
        ("discovery", f"/ctem/assessments/{assessment_id}/discovery"),
        ("priorities", f"/ctem/assessments/{assessment_id}/priorities"),
        ("validation", f"/ctem/assessments/{assessment_id}/validation"),
        ("mobilization", f"/ctem/assessments/{assessment_id}/mobilization"),
        ("report", f"/ctem/assessments/{assessment_id}/report"),
        ("executive", f"/ctem/assessments/{assessment_id}/report/executive"),
    ]:
        _, body = request("GET", path, "analyst", {200})
        mod[label] = True if data_of(body) is not None else False

    _, _ = request("POST", f"/ctem/assessments/{assessment_id}/validate", "admin", {202}, json_body={"findings": []})
    _, _ = request("POST", f"/ctem/assessments/{assessment_id}/mobilize", "admin", {202}, json_body={})

    _, body = request("GET", f"/ctem/assessments/{assessment_id}/findings", "analyst", {200}, params={"per_page": 20})
    findings = data_of(body)
    mod["finding_total"] = findings["total"]
    first_finding = first_item(findings)
    if first_finding is not None:
        _, body = request("GET", f"/ctem/findings/{first_finding['id']}", "analyst", {200})
        require(data_of(body)["id"] == first_finding["id"], "CTEM finding detail should match")
        _, body = request("PUT", f"/ctem/findings/{first_finding['id']}/status", "admin", {200}, json_body={"status": "accepted_risk", "notes": "QA acceptance"})
        mod["updated_finding_status"] = data_of(body)["status"]

    _, body = request("GET", f"/ctem/assessments/{assessment_id}/remediation-groups", "analyst", {200})
    groups = data_of(body)
    mod["remediation_group_total"] = len(groups)
    if groups:
        group_id = groups[0]["id"]
        _, body = request("GET", f"/ctem/remediation-groups/{group_id}", "analyst", {200})
        group_detail = data_of(body)
        require(group_detail["group"]["id"] == group_id, "CTEM remediation group detail should match")
        _, body = request("PUT", f"/ctem/remediation-groups/{group_id}/status", "admin", {200}, json_body={"status": "in_progress"})
        mod["group_status"] = data_of(body)["status"]

    _, body = request("POST", f"/ctem/assessments/{assessment_id}/report/export", "admin", {202}, json_body={"format": "pdf"})
    mod["report_export_status"] = data_of(body)["status"]

    _, body = request("GET", "/ctem/dashboard", "analyst", {200})
    mod["dashboard"] = data_of(body)
    _, body = request("GET", "/ctem/exposure-score", "analyst", {200})
    mod["current_exposure_score"] = data_of(body)["score"]
    _, body = request("GET", "/ctem/exposure-score/history", "analyst", {200}, params={"days": 90})
    mod["history_points"] = len(data_of(body))
    _, body = request("POST", "/ctem/exposure-score/calculate", "admin", {200})
    mod["forced_exposure_score"] = data_of(body)["score"]

    if other_assessment_id and other_assessment_id != assessment_id:
        _, body = request("GET", f"/ctem/assessments/{assessment_id}/compare/{other_assessment_id}", "analyst", {200})
        comparison = data_of(body)
        mod["comparison_direction"] = comparison["delta"]["score_direction"]

    return {"assessment_id": assessment_id}


def run_risk_and_dashboard():
    log("risk/dashboard: score, heatmap, trends, dashboard sections")
    risk_mod = module_summary("risk")
    dash_mod = module_summary("dashboard")

    _, body = request("GET", "/risk/score", "analyst", {200})
    risk_mod["score"] = data_of(body)["overall_score"]
    _, body = request("GET", "/risk/score/trend", "analyst", {200}, params={"days": 30})
    trend_data = data_of(body)
    risk_mod["trend_points"] = len(trend_data) if isinstance(trend_data, list) else 0
    _, body = request("GET", "/risk/score/recalculate", "admin", {200})
    risk_mod["recalculated_score"] = data_of(body)["overall_score"]
    _, body = request("GET", "/risk/heatmap", "analyst", {200})
    risk_mod["heatmap_rows"] = len(data_of(body)["rows"])
    _, body = request("GET", "/risk/top-risks", "analyst", {200})
    risk_mod["top_risks"] = len(data_of(body))
    _, body = request("GET", "/risk/recommendations", "analyst", {200})
    risk_mod["recommendations"] = len(data_of(body))

    for label, path in [
        ("dashboard", "/dashboard"),
        ("kpis", "/dashboard/kpis"),
        ("alerts_timeline", "/dashboard/alerts-timeline"),
        ("severity_distribution", "/dashboard/severity-distribution"),
        ("mttr", "/dashboard/mttr"),
        ("analyst_workload", "/dashboard/analyst-workload"),
        ("top_attacked_assets", "/dashboard/top-attacked-assets"),
        ("mitre_heatmap", "/dashboard/mitre-heatmap"),
        ("trends", "/dashboard/trends"),
    ]:
        _, body = request("GET", path, "analyst", {200}, params={"days": 30} if label == "trends" else None)
        payload = data_of(body)
        if label == "dashboard":
            dash_mod["cached_at"] = payload.get("cached_at")
            dash_mod["kpis"] = payload["kpis"]
        elif isinstance(payload, list):
            dash_mod[label] = len(payload)
        elif isinstance(payload, dict) and "data" in payload:
            dash_mod[label] = len(payload["data"])
        else:
            dash_mod[label] = payload


def run_remediation(primary_alert_id, web_asset_id, suffix):
    log("remediation: revision + approval + dry-run + execute + verify + close")
    mod = module_summary("remediation")
    create_payload = {
        "alert_id": primary_alert_id,
        "type": "custom",
        "severity": "high",
        "title": f"QA Custom Remediation {suffix}",
        "description": "QA governed remediation for full-suite validation",
        "plan": {
            "steps": [{"number": 1, "action": "manual_change", "description": "Perform manual remediation", "target": "qa-target", "expected": "issue mitigated"}],
            "reversible": True,
            "requires_reboot": False,
            "estimated_downtime": "0",
            "risk_level": "low",
        },
        "affected_asset_ids": [web_asset_id],
        "execution_mode": "manual",
        "requires_approval_from": "security_manager",
        "tags": ["qa", "custom", "full-suite"],
        "metadata": {"qa": True},
    }
    _, body = request("POST", "/remediation", "admin", {201}, json_body=create_payload)
    action = data_of(body)
    remediation_id = action["id"]
    mod["id"] = remediation_id

    status_code, body = request("POST", f"/remediation/{remediation_id}/execute", "admin", {400, 403}, json_body={})
    mod["execute_without_approval_status"] = status_code
    require(status_code == 403, "execute without approval must be blocked with 403", body)

    _, body = request("PUT", f"/remediation/{remediation_id}", "admin", {200}, json_body={"description": "QA governed remediation updated"})
    require(data_of(body)["description"] == "QA governed remediation updated", "remediation update should modify draft")

    _, body = request("POST", f"/remediation/{remediation_id}/submit", "admin", {200}, json_body={})
    require(data_of(body)["status"] == "pending_approval", "remediation submit should move to pending_approval")

    _, body = request("POST", f"/remediation/{remediation_id}/request-revision", "manager", {200}, json_body={"notes": "add operator note"})
    require(data_of(body)["status"] == "revision_requested", "remediation revision request should move to revision_requested")

    _, body = request("PUT", f"/remediation/{remediation_id}", "admin", {200}, json_body={"description": "QA governed remediation revised"})
    require(data_of(body)["description"] == "QA governed remediation revised", "remediation update should work after revision_requested")

    _, body = request("POST", f"/remediation/{remediation_id}/submit", "admin", {200}, json_body={})
    require(data_of(body)["status"] == "pending_approval", "remediation resubmit should return to pending_approval")

    _, body = request("POST", f"/remediation/{remediation_id}/approve", "manager", {200}, json_body={"notes": "approved for QA"})
    require(data_of(body)["status"] == "approved", "remediation approval should set approved")

    status_code, body = request("POST", f"/remediation/{remediation_id}/execute", "admin", {400, 403}, json_body={})
    mod["execute_without_dry_run_status"] = status_code
    require(status_code == 400, "execute without dry-run must be blocked with 400", body)

    _, body = request("POST", f"/remediation/{remediation_id}/dry-run", "admin", {200}, json_body={})
    dry_run = data_of(body)
    require(dry_run["success"] is True, "custom remediation dry-run should succeed")

    _, body = request("GET", f"/remediation/{remediation_id}/dry-run", "analyst", {200})
    require(data_of(body)["success"] is True, "remediation dry-run GET should return successful dry-run")

    _, body = request(
        "POST",
        f"/remediation/{remediation_id}/execute",
        "admin",
        {200},
        json_body={"manual_confirmation": MANUAL_CONFIRM},
    )
    executed = data_of(body)
    require(executed["status"] == "executed", "custom remediation execute should reach executed", executed)

    _, body = request(
        "POST",
        f"/remediation/{remediation_id}/verify",
        "admin",
        {200},
        json_body={"manual_confirmation": "Operator verified outcome"},
    )
    verified = data_of(body)
    require(verified["status"] == "verified", "custom remediation verify should reach verified", verified)

    _, body = request("GET", f"/remediation/{remediation_id}/audit-trail", "analyst", {200})
    audit = data_of(body)
    require(len(audit) >= 1, "remediation audit trail should contain entries")
    mod["audit_entries"] = len(audit)

    _, body = request("POST", f"/remediation/{remediation_id}/close", "admin", {200}, json_body={})
    require(data_of(body)["status"] == "closed", "remediation close should reach closed")

    _, body = request("GET", f"/remediation/{remediation_id}", "analyst", {200})
    mod["final_status"] = data_of(body)["status"]
    _, body = request("GET", "/remediation", "analyst", {200}, params={"per_page": 20})
    mod["list_total"] = body["total"]
    _, body = request("GET", "/remediation/stats", "analyst", {200})
    mod["stats"] = data_of(body)


def run_dspm_and_vciso():
    log("dspm/vciso: scan, summaries, briefing, report")
    dspm_mod = module_summary("dspm")
    vciso_mod = module_summary("vciso")

    _, body = request("POST", "/dspm/scan", "admin", {202}, json_body={})
    scan = data_of(body)["scan"]
    scan_id = scan["id"]
    dspm_mod["scan_id"] = scan_id

    def dspm_done():
        _, resp_body = request("GET", f"/dspm/scans/{scan_id}", "analyst", {200})
        result = data_of(resp_body)
        return result["scan"]["status"] in {"completed", "failed"}, result

    scan_result = poll("DSPM scan completion", dspm_done, timeout=240, interval=3)
    require(scan_result["scan"]["status"] == "completed", "DSPM scan should complete", scan_result)
    dspm_mod["scan"] = {
        "status": scan_result["scan"]["status"],
        "assets_scanned": scan_result["assets_scanned"],
        "findings_count": scan_result["findings_count"],
    }

    _, body = request("GET", "/dspm/data-assets", "analyst", {200}, params={"per_page": 10})
    data_assets = body
    require(data_assets["total"] >= 1, "DSPM data assets should exist after scan")
    dspm_mod["data_asset_total"] = data_assets["total"]
    first_data_asset = first_item(data_assets)
    if first_data_asset is not None:
        _, body = request("GET", f"/dspm/data-assets/{first_data_asset['id']}", "analyst", {200})
        require(data_of(body)["id"] == first_data_asset["id"], "DSPM data asset detail should match")

    _, body = request("GET", "/dspm/classification", "analyst", {200})
    dspm_mod["classification"] = data_of(body)
    _, body = request("GET", "/dspm/exposure", "analyst", {200})
    dspm_mod["exposure"] = data_of(body)
    _, body = request("GET", "/dspm/dependencies", "analyst", {200})
    dependencies = data_of(body)
    dspm_mod["dependency_nodes"] = len(dependencies.get("nodes", [])) if isinstance(dependencies, dict) else 0
    _, body = request("GET", "/dspm/dashboard", "analyst", {200})
    dspm_mod["dashboard"] = data_of(body)
    _, body = request("GET", "/dspm/scans", "analyst", {200}, params={"per_page": 10})
    dspm_mod["scan_history_total"] = body["total"]

    _, body = request("GET", "/vciso/briefing/history", "analyst", {200}, params={"per_page": 20})
    before_history = body["total"]

    _, body = request("GET", "/vciso/briefing", "analyst", {200}, params={"period_days": 30})
    briefing = data_of(body)
    require(briefing["risk_posture"] is not None, "vCISO briefing should contain risk posture")
    vciso_mod["briefing_generated"] = True

    _, body = request("GET", "/vciso/recommendations", "analyst", {200})
    vciso_mod["recommendation_count"] = len(data_of(body))

    _, body = request("GET", "/vciso/posture-summary", "analyst", {200})
    vciso_mod["posture_summary"] = data_of(body)

    _, body = request("POST", "/vciso/report", "admin", {202}, json_body={"type": "executive", "period_days": 30})
    vciso_mod["report_status"] = data_of(body)["status"]

    _, body = request("GET", "/vciso/briefing/history", "analyst", {200}, params={"per_page": 20})
    after_history = body["total"]
    require(after_history >= before_history, "vCISO briefing history should remain queryable")
    vciso_mod["history_before"] = before_history
    vciso_mod["history_after"] = after_history


def cleanup():
    log("cleanup: deleting disposable rules and assets")
    cleanup_mod = module_summary("cleanup")
    for rule_id in artifacts["rule_ids"]:
        try:
            request("DELETE", f"/rules/{rule_id}", "admin", {204})
        except Exception as exc:
            summary["errors"].append(f"cleanup rule {rule_id}: {exc}")
    cleanup_mod["deleted_rule_count"] = len(artifacts["rule_ids"])

    if artifacts["asset_ids"]:
        try:
            request("DELETE", "/assets/bulk", "admin", {204}, json_body={"asset_ids": artifacts["asset_ids"]})
            cleanup_mod["deleted_asset_count"] = len(artifacts["asset_ids"])
        except Exception as exc:
            summary["errors"].append(f"cleanup assets: {exc}")


def main():
    suffix = unique_suffix()
    fixtures = load_fixtures()
    summary["artifacts"]["fixtures"] = fixtures
    try:
        assets_ctx = run_assets(fixtures, suffix)
        run_vulnerabilities(fixtures, suffix)
        detection_ctx = run_threats_and_rules(fixtures, assets_ctx, suffix)
        run_mitre()
        ctem_ctx = run_ctem(assets_ctx, suffix)
        run_risk_and_dashboard()
        run_remediation(detection_ctx["primary_alert_id"], assets_ctx["web_asset_id"], suffix)
        run_dspm_and_vciso()
        summary["artifacts"].update(
            {
                "created_asset_ids": artifacts["asset_ids"],
                "created_rule_ids": artifacts["rule_ids"],
                "ctem_assessment_id": ctem_ctx["assessment_id"],
                "primary_alert_id": detection_ctx["primary_alert_id"],
                "feedback_alert_id": detection_ctx["feedback_alert_id"],
            }
        )
        summary["status"] = "passed"
    except Exception as exc:
        summary["status"] = "failed"
        summary["errors"].append(str(exc))
        raise
    finally:
        try:
            cleanup()
        finally:
            summary["finished_at"] = dt.datetime.utcnow().isoformat() + "Z"
            SUMMARY_PATH.write_text(json.dumps(summary, indent=2, default=str))
            log(f"summary written to {SUMMARY_PATH}")


if __name__ == "__main__":
    try:
        main()
    except Exception as exc:
        print(f"QA FAILED: {exc}", file=sys.stderr)
        sys.exit(1)
