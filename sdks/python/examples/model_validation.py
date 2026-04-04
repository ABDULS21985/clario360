from clario360 import Clario360


def main() -> None:
    client = Clario360()
    models = client.ai.models.list(per_page=10)
    for model in models:
        print(model.id, model.name)
    client.close()


if __name__ == "__main__":
    main()
