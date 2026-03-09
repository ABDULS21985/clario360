from __future__ import annotations

from pathlib import Path

from click.testing import CliRunner

from clario360_cli.cli import cli


def test_config_set_and_get(monkeypatch, tmp_path: Path) -> None:
    config_dir = tmp_path / ".clario360"
    config_path = config_dir / "config.yaml"
    monkeypatch.setattr("clario360_cli.commands.config.CONFIG_DIR", config_dir)
    monkeypatch.setattr("clario360_cli.commands.config.CONFIG_PATH", config_path)

    runner = CliRunner()
    set_result = runner.invoke(cli, ["config", "set", "api-url", "https://api.example.test"])
    get_result = runner.invoke(cli, ["config", "get", "api-url"])

    assert set_result.exit_code == 0
    assert get_result.exit_code == 0
    assert "https://api.example.test" in get_result.output
