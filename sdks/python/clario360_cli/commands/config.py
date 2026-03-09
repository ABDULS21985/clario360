from __future__ import annotations

from pathlib import Path
from typing import Any, Dict, Optional

import click

CONFIG_DIR = Path.home() / ".clario360"
CONFIG_PATH = CONFIG_DIR / "config.yaml"
SUPPORTED_KEYS = {"api-url", "api-key", "email", "password"}


def _yaml_module() -> Any:
    try:
        import yaml
    except ImportError as exc:
        raise click.ClickException("pyyaml is required for config commands. Install clario360[cli].") from exc
    return yaml


def load_settings() -> Dict[str, str]:
    if not CONFIG_PATH.exists():
        return {}
    yaml = _yaml_module()
    loaded = yaml.safe_load(CONFIG_PATH.read_text(encoding="utf-8"))
    if isinstance(loaded, dict):
        return {str(key): str(value) for key, value in loaded.items() if value is not None}
    return {}


def save_settings(settings: Dict[str, str]) -> None:
    yaml = _yaml_module()
    CONFIG_DIR.mkdir(parents=True, exist_ok=True)
    CONFIG_PATH.write_text(yaml.safe_dump(settings, sort_keys=True), encoding="utf-8")


def get_setting(key: str) -> Optional[str]:
    return load_settings().get(key)


@click.group("config")
def config_group() -> None:
    """Manage local CLI configuration."""


@config_group.command("set")
@click.argument("key")
@click.argument("value")
def set_config(key: str, value: str) -> None:
    """Persist a CLI setting under ~/.clario360/config.yaml."""
    normalized = key.lower()
    if normalized not in SUPPORTED_KEYS:
        raise click.ClickException(f"Unsupported config key: {key}")
    settings = load_settings()
    settings[normalized] = value
    save_settings(settings)
    click.echo(f"Saved {normalized} to {CONFIG_PATH}")


@config_group.command("get")
@click.argument("key", required=False)
def get_config(key: Optional[str]) -> None:
    """Read one config value or dump the whole CLI config."""
    settings = load_settings()
    if key is None:
        if not settings:
            click.echo("No CLI settings configured.")
            return
        for name, value in settings.items():
            masked = _mask_secret(name, value)
            click.echo(f"{name}: {masked}")
        return

    normalized = key.lower()
    if normalized not in SUPPORTED_KEYS:
        raise click.ClickException(f"Unsupported config key: {key}")
    value = settings.get(normalized)
    if value is None:
        raise click.ClickException(f"{normalized} is not configured.")
    click.echo(_mask_secret(normalized, value))


def _mask_secret(key: str, value: str) -> str:
    if key in {"api-key", "password"} and len(value) > 8:
        return f"{value[:4]}...{value[-4:]}"
    if key in {"api-key", "password"}:
        return "********"
    return value
