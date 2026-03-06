export default function DashboardHome() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">Dashboard</h1>
        <p className="text-muted-foreground">
          Welcome to Clario 360 Enterprise AI Platform
        </p>
      </div>

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        {[
          { title: "Cybersecurity", description: "Active alerts and threat monitoring", count: "—" },
          { title: "Data Suite", description: "Data pipelines and quality metrics", count: "—" },
          { title: "Acta", description: "Document management and signatures", count: "—" },
          { title: "Visus360", description: "Dashboards and visual reports", count: "—" },
        ].map((card) => (
          <div
            key={card.title}
            className="rounded-lg border bg-card p-6 shadow-sm"
          >
            <h3 className="text-sm font-medium text-muted-foreground">
              {card.title}
            </h3>
            <p className="mt-2 text-3xl font-bold">{card.count}</p>
            <p className="mt-1 text-xs text-muted-foreground">
              {card.description}
            </p>
          </div>
        ))}
      </div>
    </div>
  );
}
