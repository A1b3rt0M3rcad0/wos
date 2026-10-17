"""A fresh process discovers the Outcome using only endpoint, Tenant and token."""
import json
import os
from urllib.parse import urlencode
from urllib.request import Request, urlopen


def get(path):
    req = Request(os.environ["WOS_READER_URL"] + path, headers={"Authorization": "Bearer " + os.environ["WOS_READER_TOKEN"]})
    with urlopen(req, timeout=15) as response:
        return json.load(response)


tenant = os.environ["WOS_READER_TENANT"]
page = get(f"/tenants/{tenant}/outcomes?" + urlencode({"text": "Full shared acceptance"}))
assert len(page["items"]) == 1, "new consumer did not discover the fixture Outcome"
state = get(f"/tenants/{tenant}/outcomes/{page['items'][0]['id']}/state")
print(json.dumps(state))
