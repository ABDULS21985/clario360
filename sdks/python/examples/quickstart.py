from clario360 import Clario360


def main() -> None:
    client = Clario360()
    alerts = client.cyber.alerts.list(per_page=5)
    for alert in alerts:
        print(f"{alert.id}: {alert.title} [{alert.severity}]")
    client.close()


if __name__ == "__main__":
    main()
