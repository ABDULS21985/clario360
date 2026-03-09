from __future__ import annotations

from dataclasses import dataclass
from typing import Any, Optional

import click

from clario360 import Clario360
from clario360_cli.commands import alerts_group, assets_group, config_group, pipelines_group, risk_group
from clario360_cli.commands.config import get_setting
from clario360_cli.formatters import render_csv, render_json, render_table


@dataclass
class CLIState:
    output: str
    api_url: Optional[str]
    api_key: Optional[str]
    email: Optional[str]
    password: Optional[str]
    _client: Optional[Clario360] = None

    def build_client(self) -> Clario360:
        if self._client is None:
            self._client = Clario360(
                api_url=self.api_url or get_setting("api-url"),
                api_key=self.api_key or get_setting("api-key"),
                email=self.email or get_setting("email"),
                password=self.password or get_setting("password"),
            )
        return self._client

    def emit(self, value: Any) -> None:
        if self.output == "json":
            click.echo(render_json(value))
            return
        if self.output == "csv":
            click.echo(render_csv(value))
            return
        click.echo(render_table(value))

    def close(self) -> None:
        if self._client is not None:
            self._client.close()
            self._client = None


@click.group()
@click.option("--api-url", type=str, default=None, help="Override the configured API URL.")
@click.option("--api-key", type=str, default=None, help="Override the configured API key.")
@click.option("--email", type=str, default=None, help="Email for password-based auth.")
@click.option("--password", type=str, default=None, help="Password for password-based auth.")
@click.option(
    "--output",
    type=click.Choice(["table", "json", "csv"], case_sensitive=False),
    default="table",
    show_default=True,
)
@click.pass_context
def cli(
    ctx: click.Context,
    api_url: str | None,
    api_key: str | None,
    email: str | None,
    password: str | None,
    output: str,
) -> None:
    """Official command-line interface for the Clario 360 platform."""
    ctx.obj = CLIState(
        output=output,
        api_url=api_url,
        api_key=api_key,
        email=email,
        password=password,
    )


@cli.result_callback()
@click.pass_context
def close_client(ctx: click.Context, _: Any, **__: Any) -> None:
    state = ctx.obj
    if isinstance(state, CLIState):
        state.close()


cli.add_command(alerts_group)
cli.add_command(assets_group)
cli.add_command(config_group)
cli.add_command(pipelines_group)
cli.add_command(risk_group)


if __name__ == "__main__":
    cli()
