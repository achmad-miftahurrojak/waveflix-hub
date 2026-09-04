export default function Loading() {
  return (
    <div className="min-h-screen animate-pulse">
      {}
      <div className="h-[75vh] min-h-[500px] w-full bg-white/5" />

      {}
      <div className="px-[4%] py-8 space-y-12">
        {Array.from({ length: 3 }).map((_, i) => (
          <div key={i}>
            <div className="mb-4 h-8 w-48 rounded bg-white/10" />
            <div className="flex gap-4 overflow-hidden">
              {Array.from({ length: 6 }).map((_, j) => (
                <div
                  key={j}
                  className="h-[240px] w-[160px] shrink-0 rounded-md bg-white/10"
                />
              ))}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
