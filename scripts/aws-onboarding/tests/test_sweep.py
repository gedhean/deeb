#!/usr/bin/env python3
"""
Fast logic tests for the multi-account onboarder — no AWS, no DB, no network.

Stubs the heavy third-party modules (boto3/psycopg2/pymysql/requests) and exercises the
pure logic: config parsing/validation, alias defaulting, assume-role vs ambient session
selection, cross-account name disambiguation + skip, and project assignment.

Run:  python3 tests/test_sweep.py     (needs PyYAML; `pip install -r requirements-aws.txt`)
"""

import importlib.util
import os
import sys
import tempfile
import types

HERE = os.path.dirname(os.path.abspath(__file__))
SCRIPTS = os.path.dirname(HERE)


def _install_stubs():
    """Stub the heavy deps so the onboarder module imports without them installed."""
    for name in ("pymysql", "requests"):
        sys.modules[name] = types.ModuleType(name)

    boto3 = types.ModuleType("boto3")

    class FakeSTS:
        def __init__(self):
            self.calls = []

        def assume_role(self, RoleArn, RoleSessionName):  # noqa: N803 (boto3 kwarg names)
            self.calls.append((RoleArn, RoleSessionName))
            return {"Credentials": {"AccessKeyId": "AK", "SecretAccessKey": "SK", "SessionToken": "TK"}}

    class FakeSession:
        def __init__(self, **kwargs):
            self.kwargs = kwargs

        def client(self, name, region_name=None):
            return ("client", name, region_name, self.kwargs)

    sts = FakeSTS()
    boto3.client = lambda name, **kw: sts if name == "sts" else ("client", name, kw)
    boto3.Session = FakeSession
    sys.modules["boto3"] = boto3

    psy = types.ModuleType("psycopg2")
    psy.sql = types.ModuleType("psycopg2.sql")
    ext = types.ModuleType("psycopg2.extensions")
    ext.ISOLATION_LEVEL_AUTOCOMMIT = 0
    sys.modules["psycopg2"] = psy
    sys.modules["psycopg2.sql"] = psy.sql
    sys.modules["psycopg2.extensions"] = ext
    return sts


def _load_module():
    sys.path.insert(0, SCRIPTS)  # so `import discover` resolves
    spec = importlib.util.spec_from_file_location("onboarder", os.path.join(SCRIPTS, "aws-onboard-to-deeb.py"))
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


class FakeResp:
    def __init__(self, data):
        self._d = data

    def raise_for_status(self):
        pass

    def json(self):
        return self._d


def test_extract_list(m):
    assert m._extract_list([{"a": 1}]) == [{"a": 1}]
    assert m._extract_list({"database_instances": [1]}, "database_instances", "data") == [1]
    try:
        m._extract_list({"nope": []}, "x")
        raise AssertionError("expected ValueError")
    except ValueError:
        pass


def test_composed_name(m):
    assert m.composed_name("app-db", "prod") == "app-db (prod)"


def test_normalize_account(m):
    default_filters = {"Environment": ["staging"]}
    acc = m.normalize_account(0, {"account_id": "111", "regions": ["us-east-1"],
                                  "environment": "Production", "project": "Proj Production"},
                              default_filters)
    assert acc["alias"] == "111" and acc["role_arn"] == "" and acc["regions"] == ["us-east-1"]
    assert acc["tag_filters"] == default_filters   # inherits the default
    # per-account override wins
    acc2 = m.normalize_account(0, {"account_id": "1", "regions": ["us-east-1"],
                                   "environment": "E", "project": "P", "tag_filters": {}},
                               default_filters)
    assert acc2["tag_filters"] == {}
    for bad in [{}, {"account_id": "1"}, {"account_id": "1", "environment": "E", "project": "P"},
                {"account_id": "1", "environment": "E", "project": "P", "regions": []}]:
        try:
            m.normalize_account(0, bad, default_filters)
            raise AssertionError(f"expected ConfigError for {bad}")
        except m.ConfigError:
            pass


def test_matches_tag_filters(m):
    # empty filter matches everything
    assert m.matches_tag_filters({}, {})
    assert m.matches_tag_filters({"Environment": "production"}, {})
    f = {"Environment": ["prod", "production", "staging", "homol", "tooling"]}
    assert m.matches_tag_filters({"Environment": "production"}, f)
    assert m.matches_tag_filters({"Environment": "PROD"}, f)          # case-insensitive value
    assert not m.matches_tag_filters({"Environment": "sandbox"}, f)   # value not allowed
    assert not m.matches_tag_filters({}, f)                          # missing tag
    assert not m.matches_tag_filters({"environment": "prod"}, f)     # key case-sensitive
    # multiple keys: all must match
    multi = {"Environment": ["prod"], "Team": ["payments"]}
    assert m.matches_tag_filters({"Environment": "prod", "Team": "payments"}, multi)
    assert not m.matches_tag_filters({"Environment": "prod"}, multi)


def test_normalize_tag_filters(m):
    assert m.normalize_tag_filters("x", None) == {}
    assert m.normalize_tag_filters("x", {"Environment": "prod"}) == {"Environment": ["prod"]}
    assert m.normalize_tag_filters("x", {"Environment": ["a", "b"]}) == {"Environment": ["a", "b"]}
    for bad in ["notamap", {"Environment": []}, {"Environment": None}]:
        try:
            m.normalize_tag_filters("x", bad)
            raise AssertionError(f"expected ConfigError for {bad}")
        except m.ConfigError:
            pass


def test_load_config(m):
    cfg_file = tempfile.NamedTemporaryFile("w", suffix=".yaml", delete=False)
    cfg_file.write(
        """
accounts:
  - account_id: "111"
    alias: prod
    role_arn: arn:aws:iam::111:role/deeb-onboarder
    regions: [us-east-1]
    environment: Production
    project: Proj Production
  - account_id: "222"
    regions: [eu-west-1]
    environment: Staging
    project: Proj Staging
"""
    )
    cfg_file.close()
    for k in ("DEEB_API_URL", "DEEB_API_TOKEN"):
        os.environ.pop(k, None)
    os.environ["DEEB_ONBOARDER_CONFIG"] = cfg_file.name
    try:
        m.load_config()
        raise AssertionError("expected ConfigError for missing env vars")
    except m.ConfigError as e:
        assert "DEEB_API_URL" in str(e)
    os.environ.update({"DEEB_API_URL": "http://x/", "DEEB_API_TOKEN": "deeb_pat_t"})
    cfg = m.load_config()
    assert cfg["deeb_api_url"] == "http://x"
    assert len(cfg["accounts"]) == 2
    assert cfg["accounts"][1]["alias"] == "222"   # defaulted to account_id
    assert cfg["accounts"][1]["role_arn"] == ""   # ambient creds
    # no tag_filters in config => built-in default applies to every account
    assert cfg["accounts"][0]["tag_filters"] == m.DEFAULT_TAG_FILTERS
    return cfg


def test_load_config_tag_filters(m):
    cfg_file = tempfile.NamedTemporaryFile("w", suffix=".yaml", delete=False)
    cfg_file.write(
        """
tag_filters:
  Environment: [production, tooling]
accounts:
  - account_id: "111"
    regions: [us-east-1]
    environment: Production
    project: Proj Production
  - account_id: "222"
    regions: [us-east-1]
    environment: Staging
    project: Proj Staging
    tag_filters: {}
"""
    )
    cfg_file.close()
    os.environ.update({"DEEB_API_URL": "http://x/", "DEEB_API_TOKEN": "deeb_pat_t"})
    os.environ["DEEB_ONBOARDER_CONFIG"] = cfg_file.name
    cfg = m.load_config()
    assert cfg["accounts"][0]["tag_filters"] == {"Environment": ["production", "tooling"]}  # top-level
    assert cfg["accounts"][1]["tag_filters"] == {}                                          # per-account override


def test_session_for(m, sts, cfg):
    ob = object.__new__(m.DeebOnboarder)
    ob.sts = sts
    s1 = ob.session_for(cfg["accounts"][0])          # role_arn set -> assume
    assert sts.calls[-1] == ("arn:aws:iam::111:role/deeb-onboarder", "deeb-onboarder")
    assert s1.kwargs.get("aws_access_key_id") == "AK"
    before = len(sts.calls)
    s2 = ob.session_for(cfg["accounts"][1])          # role_arn empty -> ambient
    assert len(sts.calls) == before and s2.kwargs == {}


def test_disambiguation_and_assignment(m):
    class FakeReqSession:
        def __init__(self):
            self.puts = []

        def get(self, url, params=None):
            if "/environments" in url:
                return FakeResp({"environments": [{"id": "env-prod", "name": "Production"},
                                                  {"id": "env-stg", "name": "Staging"}]})
            if "/projects" in url:
                return FakeResp({"projects": [{"id": "prj-prod", "name": "Proj Production"},
                                              {"id": "prj-stg", "name": "Proj Staging"}]})
            if "/database-instances" in url:
                off = (params or {}).get("offset", 0)
                return FakeResp({"database_instances": [{"name": "app-db (prod)"}] if off == 0 else []})
            if "/databases" in url:
                return FakeResp({"databases": [{"id": "d1", "project_id": None, "environment_id": None}]})
            raise AssertionError(url)

        def put(self, url, json=None):
            self.puts.append((url, json))
            return FakeResp({})

    ob = object.__new__(m.DeebOnboarder)
    ob.deeb_api_url = "http://x"
    ob.dry_run = False
    ob.session = FakeReqSession()

    assert ob.resolve_environment_id("Production") == "env-prod"
    assert ob.resolve_project_id("Proj Staging") == "prj-stg"
    reg = ob.registered_instance_names()
    assert reg == {"app-db (prod)"}
    # both accounts have a DB named "app-db"; only staging's composed name is new
    assert m.composed_name("app-db", "prod") in reg
    assert m.composed_name("app-db", "staging") not in reg
    # assign only the unassigned db
    n = ob.assign_databases_to_project("newinst", "prj-stg", "env-stg")
    assert n == 1 and len(ob.session.puts) == 1


def main():
    sts = _install_stubs()
    m = _load_module()
    test_extract_list(m)
    test_composed_name(m)
    test_normalize_account(m)
    test_matches_tag_filters(m)
    test_normalize_tag_filters(m)
    cfg = test_load_config(m)
    test_load_config_tag_filters(m)
    test_session_for(m, sts, cfg)
    test_disambiguation_and_assignment(m)
    print("ALL MULTI-ACCOUNT LOGIC TESTS PASSED")


if __name__ == "__main__":
    main()
