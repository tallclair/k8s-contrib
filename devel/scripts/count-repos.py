import requests

repos = [
    'kubernetes',
    'kubernetes-client',
    'kubernetes-csi',
    'kubernetes-extensions',
    'kubernetes-federation',
    'kubernetes-incubator',
    'kubernetes-sidecars',
    'kubernetes-sig-testing',
    'kubernetes-sigs',
    'kubernetes-test',
    'kubernetes-tools',
]

for repo in repos:
    resp = requests.get(url="https://api.github.com/orgs/" + repo + "/repos?per_page=200")
    data = resp.json()

    for item in data:
        name = item['full_name'].split('/')[1]
        print("%s, %s" % (repo + "/" + name, item["created_at"]))
