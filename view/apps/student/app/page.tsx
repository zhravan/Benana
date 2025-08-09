import { Button } from "@workspace/ui/components/button"

export default function Page() {
  return (
    <div className="flex items-center justify-center min-h-svh">
      <div className="flex flex-col items-center justify-center gap-4">
        <h1 className="text-2xl font-bold">Student Portal</h1>
        <p className="text-muted-foreground">Welcome to the Benana Student Portal</p>
        <Button size="sm">Access Courses</Button>
      </div>
    </div>
  )
}
