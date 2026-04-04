from clario360 import Clario360


def main() -> None:
    client = Clario360()
    pipelines = client.data.pipelines.list(per_page=10)
    for pipeline in pipelines:
        print(f"{pipeline.name}: {pipeline.status}")
    if pipelines.data:
        run = client.data.pipelines.run(pipelines.data[0].id)
        print(f"Started run {run.id} with status {run.status}")
    client.close()


if __name__ == "__main__":
    main()
