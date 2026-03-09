from __future__ import annotations

import click


@click.group("risk")
def risk_group() -> None:
    """Retrieve cyber risk views."""


@risk_group.command("score")
@click.pass_obj
def risk_score(state: "CLIState") -> None:
    client = state.build_client()
    state.emit(client.cyber.risk.score())


@risk_group.command("heatmap")
@click.pass_obj
def risk_heatmap(state: "CLIState") -> None:
    client = state.build_client()
    state.emit(client.cyber.risk.heatmap())
