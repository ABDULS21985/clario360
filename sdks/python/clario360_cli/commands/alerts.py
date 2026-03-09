from __future__ import annotations

import click


@click.group("alerts")
def alerts_group() -> None:
    """Work with cybersecurity alerts."""


@alerts_group.command("list")
@click.option("--severity", type=str, default=None, help="Filter by severity.")
@click.option("--status", type=str, default=None, help="Filter by status.")
@click.option("--search", type=str, default=None, help="Full-text search string.")
@click.option("--page", type=int, default=1, show_default=True)
@click.option("--per-page", type=int, default=25, show_default=True)
@click.pass_obj
def list_alerts(state: "CLIState", severity: str | None, status: str | None, search: str | None, page: int, per_page: int) -> None:
    client = state.build_client()
    result = client.cyber.alerts.list(
        severity=severity,
        status=status,
        search=search,
        page=page,
        per_page=per_page,
    )
    state.emit(result)


@alerts_group.command("get")
@click.argument("alert_id")
@click.pass_obj
def get_alert(state: "CLIState", alert_id: str) -> None:
    client = state.build_client()
    state.emit(client.cyber.alerts.get(alert_id))


@alerts_group.command("ack")
@click.argument("alert_id")
@click.pass_obj
def acknowledge_alert(state: "CLIState", alert_id: str) -> None:
    client = state.build_client()
    state.emit(client.cyber.alerts.acknowledge(alert_id))


@alerts_group.command("close")
@click.argument("alert_id")
@click.option("--resolution", default="resolved", show_default=True)
@click.pass_obj
def close_alert(state: "CLIState", alert_id: str, resolution: str) -> None:
    client = state.build_client()
    state.emit(client.cyber.alerts.close(alert_id, resolution=resolution))

