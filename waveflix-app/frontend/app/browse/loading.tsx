export default function Loading() {
  return (
    <div className="min-h-screen animate-pulse pb-16">
      {}
      <div className="h-[75vh] min-h-[500px] w-full bg-white/5" />

      {}
      <div className="px-[4%] pt-8">
        <div className="mb-6 h-8 w-64 rounded bg-white/10" />
        <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6">
          {Array.from({ length: 18 }).map((_, i) => (
            <div
              key={i}
              className="aspect-[2/3] w-full rounded-md bg-white/10"
            />
          ))}
        </div>
      </div>
    </div>
  );
}
