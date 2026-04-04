from clario360 import Clario360


def main() -> None:
    client = Clario360()
    profiles = client.cyber.ueba.profiles(per_page=10)
    for profile in profiles:
        print(profile.id, getattr(profile, "risk_score", None))
    client.close()


if __name__ == "__main__":
    main()
