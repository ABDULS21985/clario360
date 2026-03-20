import csv
from pathlib import Path

from clario360 import Clario360


def load_assets(csv_path: Path) -> list[dict[str, str]]:
    with csv_path.open("r", encoding="utf-8", newline="") as handle:
        reader = csv.DictReader(handle)
        return [dict(row) for row in reader]


def main() -> None:
    client = Clario360()
    assets = load_assets(Path("assets.csv"))
    created = client.cyber.assets.bulk_create(assets)
    print(f"Created {len(created)} assets.")
    client.close()


if __name__ == "__main__":
    main()
