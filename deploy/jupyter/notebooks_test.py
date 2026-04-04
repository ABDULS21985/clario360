import json
import re
from pathlib import Path


NOTEBOOK_DIR = Path(__file__).parent / "notebooks"


def _load_notebooks():
    return sorted(NOTEBOOK_DIR.glob("*.ipynb"))


def _parse_notebook(path: Path):
    with path.open("r", encoding="utf-8") as handle:
        return json.load(handle)


def test_all_notebooks_valid_json():
    for notebook_path in _load_notebooks():
        _parse_notebook(notebook_path)


def test_all_notebooks_have_kernel():
    for notebook_path in _load_notebooks():
        notebook = _parse_notebook(notebook_path)
        assert notebook["metadata"]["kernelspec"]["name"] == "python3"


def test_all_notebooks_have_markdown():
    for notebook_path in _load_notebooks():
        notebook = _parse_notebook(notebook_path)
        markdown_cells = [cell for cell in notebook["cells"] if cell["cell_type"] == "markdown"]
        assert len(markdown_cells) >= 3, notebook_path.name


def test_all_notebooks_no_outputs():
    for notebook_path in _load_notebooks():
        notebook = _parse_notebook(notebook_path)
        for cell in notebook["cells"]:
            if cell["cell_type"] == "code":
                assert cell.get("outputs", []) == [], notebook_path.name
                assert cell.get("execution_count") is None, notebook_path.name


def test_all_notebooks_no_secrets():
    secret_patterns = [
        re.compile(r"api[_-]?key", re.IGNORECASE),
        re.compile(r"secret", re.IGNORECASE),
        re.compile(r"""password\s*=\s*['"](?!open\()""", re.IGNORECASE),
        re.compile(r"""token\s*=\s*['"]eyJ""", re.IGNORECASE),
    ]
    for notebook_path in _load_notebooks():
        notebook = _parse_notebook(notebook_path)
        content = "\n".join("".join(cell.get("source", [])) for cell in notebook["cells"])
        for pattern in secret_patterns:
            assert not pattern.search(content), f"{notebook_path.name} matched {pattern.pattern}"


def test_notebook_01_cells():
    notebook = _parse_notebook(NOTEBOOK_DIR / "01_threat_detection_quickstart.ipynb")
    content = "\n".join("".join(cell.get("source", [])) for cell in notebook["cells"])
    assert "Client.from_env()" in content
    assert "sdk.cyber.alerts.list" in content
    assert ".to_csv(" in content


def test_notebook_05_cells():
    notebook = _parse_notebook(NOTEBOOK_DIR / "05_model_validation_framework.ipynb")
    content = "\n".join("".join(cell.get("source", [])) for cell in notebook["cells"])
    assert "sdk.ai.models.get_by_name" in content
    assert "confusion_matrix" in content
    assert "roc_curve" in content
    assert "sdk.ai.models.create_version" in content


def test_notebook_08_spark():
    notebook = _parse_notebook(NOTEBOOK_DIR / "08_spark_large_scale_analysis.ipynb")
    content = "\n".join("".join(cell.get("source", [])) for cell in notebook["cells"])
    assert "SPARK_ENABLED" in content
    assert "SparkSession" in content
    assert "CLICKHOUSE_JDBC_JAR" in content
    assert "com.clickhouse.jdbc.ClickHouseDriver" in content
