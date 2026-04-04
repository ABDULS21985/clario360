from __future__ import annotations

from typing import TYPE_CHECKING, Dict, Optional

import click

if TYPE_CHECKING:
    from clario360_cli.cli import CLIState


@click.group("assets")
def assets_group() -> None:
    """Work with cyber assets."""


@assets_group.command("list")
@click.option("--type", "asset_type", type=str, default=None, help="Filter by asset type.")
@click.option("--criticality", type=str, default=None, help="Filter by criticality.")
@click.option("--status", type=str, default=None, help="Filter by status.")
@click.option("--search", type=str, default=None, help="Search assets.")
@click.option("--page", type=int, default=1, show_default=True)
@click.option("--per-page", type=int, default=25, show_default=True)
@click.pass_obj
def list_assets(
    state: "CLIState",
    asset_type: Optional[str],
    criticality: Optional[str],
    status: Optional[str],
    search: Optional[str],
    page: int,
    per_page: int,
) -> None:
    client = state.build_client()
    result = client.cyber.assets.list(
        asset_type=asset_type,
        criticality=criticality,
        status=status,
        search=search,
        page=page,
        per_page=per_page,
    )
    state.emit(result)


@assets_group.command("get")
@click.argument("asset_id")
@click.pass_obj
def get_asset(state: "CLIState", asset_id: str) -> None:
    client = state.build_client()
    state.emit(client.cyber.assets.get(asset_id))


@assets_group.command("scan")
@click.option("--cidr", type=str, default=None, help="CIDR range to scan.")
@click.option("--target", type=str, default=None, help="Single target to scan.")
@click.option("--name", type=str, default=None, help="Optional scan label.")
@click.pass_obj
def scan_assets(state: "CLIState", cidr: Optional[str], target: Optional[str], name: Optional[str]) -> None:
    payload: Dict[str, object] = {
        key: value
        for key, value in {"cidr": cidr, "target": target, "name": name}.items()
        if value is not None
    }
    if not payload:
        raise click.ClickException("Provide at least one scan parameter, such as --cidr or --target.")
    client = state.build_client()
    state.emit(client.cyber.assets.scan(payload))
