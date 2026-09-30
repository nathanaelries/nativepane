"""Standard-library HTTP and independent ZIP/XML verification against a running server."""
import io
import json
import os
import time
import urllib.request
import zipfile
import xml.etree.ElementTree as ET
from pathlib import Path

BASE = os.environ.get("NATIVEPANE_URL", "http://127.0.0.1:8080")
KEY = os.environ.get("API_KEY", "")

def request(path, method="GET", data=None, token=None):
    headers = {}
    credential = KEY if token is None else token
    if credential:
        headers["Authorization"] = "Bearer " + credential
    if isinstance(data, dict):
        data = json.dumps(data).encode()
        headers["Content-Type"] = "application/json"
    req = urllib.request.Request(BASE + path, data=data, headers=headers, method=method)
    with urllib.request.urlopen(req, timeout=30) as response:
        return response.read()

def edit_file(name, source):
    access = json.loads(request("/api/v1/sessions", "POST", {"filename": name, "mode": "edit"}))
    route = "/api/v1/sessions/" + access["id"]
    try:
        request(route + "/file", "PUT", source)
        opened = json.loads(request(route + "/document", token=access["token"]))
        first = next(f for b in opened["model"]["blocks"] for f in b["fields"] if not f.get("readOnly"))
        request(route + "/document", "PATCH", {"revision": opened["revision"], "edits": [{"id": first["id"], "text": "Verified edit & <round trip>"}]}, access["token"])
        output = request(route + "/file", token=access["token"])
        with zipfile.ZipFile(io.BytesIO(source)) as before, zipfile.ZipFile(io.BytesIO(output)) as after:
            assert set(before.namelist()) == set(after.namelist())
            changed = [n for n in before.namelist() if before.read(n) != after.read(n)]
            assert len(changed) == 1, changed
            for name in after.namelist():
                if name.endswith((".xml", ".rels")):
                    ET.fromstring(after.read(name))
            assert b"Verified edit &amp; &lt;round trip&gt;" in after.read(changed[0])
        request(route + "/events", "POST", {"type": "closed"}, access["token"])
        events = json.loads(request(route + "/events"))
        assert {"uploaded", "saved", "downloaded", "closed"} <= {e["type"] for e in events["events"]}
        return output
    finally:
        request(route, "DELETE")

if __name__ == "__main__":
    for attempt in range(30):
        try:
            request("/healthz")
            break
        except OSError:
            time.sleep(1)
    else:
        raise RuntimeError("NativePane did not start")
    spec = json.loads(request("/openapi.json"))
    assert spec["openapi"] == "3.1.0"
    for extension in ("docx", "xlsx", "pptx"):
        sample = Path(__file__).resolve().parent.parent / "samples" / ("welcome." + extension)
        edit_file(sample.name, sample.read_bytes())
        print(extension.upper() + ": HTTP edit, native download, XML integrity, preserved parts, lifecycle events PASS")
