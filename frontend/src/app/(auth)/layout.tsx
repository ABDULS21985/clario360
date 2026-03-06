export default function AuthLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="flex min-h-screen items-center justify-center bg-gradient-to-br from-primary/5 via-background to-secondary/5">
      <div className="w-full max-w-md space-y-8 rounded-xl border bg-card p-8 shadow-lg">
        <div className="text-center">
          <h1 className="text-2xl font-bold tracking-tight text-foreground">
            Clario 360
          </h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Enterprise AI Platform
          </p>
        </div>
        {children}
      </div>
    </div>
  );
}
