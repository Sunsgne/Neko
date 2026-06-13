import { Card } from "@/components/ui";
import { Construction } from "lucide-react";

export function ComingSoon({ title, desc, epic }: { title: string; desc: string; epic: string }) {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">{title}</h1>
        <p className="mt-1 text-sm text-muted">{desc}</p>
      </div>
      <Card className="flex flex-col items-center justify-center gap-3 py-16 text-center">
        <div className="grid h-12 w-12 place-items-center rounded-xl bg-primary/15 text-primary">
          <Construction className="h-6 w-6" />
        </div>
        <p className="text-sm text-muted">该模块正在持续开发中</p>
        <p className="font-mono text-xs text-muted">见 docs/TASKS.md · {epic}</p>
      </Card>
    </div>
  );
}
