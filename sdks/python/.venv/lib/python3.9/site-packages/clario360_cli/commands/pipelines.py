from __future__ import annotations

from typing import TYPE_CHECKING, Optional

import click

if TYPE_CHECKING:
    from clario360_cli.cli import CLIState


@click.group("pipelines")
def pipelines_group() -> None:
    """Manage data pipelines."""


@pipelines_group.command("list")
@click.option("--search", type=str, default=None, help="Search pipelines by name.")
@click.option("--page", type=int, default=1, show_default=True)
@click.option("--per-page", type=int, default=25, show_default=True)
@click.pass_obj
def list_pipelines(state: "CLIState", search: Optional[str], page: int, per_page: int) -> None:
    client = state.build_client()
    state.emit(client.data.pipelines.list(search=search, page=page, per_page=per_page))


@pipelines_group.command("run")
@click.argument("pipeline_id")
@click.pass_obj
def run_pipeline(state: "CLIState", pipeline_id: str) -> None:
    client = state.build_client()
    state.emit(client.data.pipelines.run(pipeline_id))
