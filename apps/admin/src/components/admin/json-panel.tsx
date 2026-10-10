"use client";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

export function JsonPanel({
  title,
  data,
}: {
  title: string;
  data: unknown;
}) {
  if (data == null) return null;
  let body = "";
  try {
    body = JSON.stringify(data, null, 2);
  } catch {
    body = "";
  }
  if (!body) return null;
  return (
    <Card>
      {title ? (
        <CardHeader>
          <CardTitle>{title}</CardTitle>
        </CardHeader>
      ) : null}
      <CardContent>
        <pre className="max-h-[640px] overflow-auto rounded-md bg-muted p-4 text-xs text-muted-foreground">
          {body}
        </pre>
      </CardContent>
    </Card>
  );
}
