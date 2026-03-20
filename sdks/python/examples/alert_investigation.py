from clario360 import Clario360


def main() -> None:
    client = Clario360()
    alerts = client.cyber.alerts.list(severity="critical", per_page=1)
    if not alerts.data:
        print("No critical alerts found.")
        client.close()
        return

    alert = client.cyber.alerts.get(alerts.data[0].id)
    print(f"Title: {alert.title}")
    print(f"Status: {alert.status}")
    if alert.explanation is not None:
        print(f"Summary: {alert.explanation.summary}")
        for action in alert.explanation.recommended_actions:
            print(f"- {action}")
    client.close()


if __name__ == "__main__":
    main()
