import {
  QueryClient,
  QueryClientProvider,
  useQuery,
} from "@tanstack/react-query";
import "./App.css";
import axios from "axios";

const queryClient = new QueryClient();

declare global {
  interface Window {
    __TANSTACK_QUERY_CLIENT__: import("@tanstack/query-core").QueryClient;
  }
}

window.__TANSTACK_QUERY_CLIENT__ = queryClient;

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <div>
      <TopBar/>
        <Feed />
      </div>
    </QueryClientProvider>
  );
}

export default App;

export function ImageCard({
  imageName,
  title,
  tags,
}: {
    imageName: string;
  title: string,
  tags: string[];
}) {
  const imageUrl = `/images/${imageName}`;

  return (
    <div className="flex flex-col gap-1 border-4">
      <div className="flex flex-row w-full basis-full gap-1">
        <p>{ title}</p>
        {tags.map((t) => (
          <TagBubble title={t} />
        ))}
      </div>
      <img src={imageUrl} />
    </div>
  );
}

export function TagBubble({ title }: { title: string }) {
  return <p className="bg-gray-300 text-gray-900 rounded-sm p-1">#{title}</p>;
}

export function TopBar() {
  return <div className="w-full h-10 bg-gray-400 border-b-2 border-gray-950">

  </div>
}

export function Feed() {
  const { isPending, isError, data, error } = useQuery({
    queryKey: ["latest"],
    queryFn: getLatest,
  });

  if (isPending) return <span>Loading...</span>;
  if (isError) return <span>Error: {error.message}</span>;

  console.log(data);

  return (
    <div className="mx-5" >
    <ul className="flex flex-col gap-3">
      {data.map((name) => (
        <li key={name}>
          <ImageCard title="some-title" tags={["foo", "bar"]} imageName={name} />
        </li>
      ))}
      </ul>
    </div>
  );
}

async function getLatest(): Promise<string[]> {
  const { data } = await axios.get<{ filenames: string[] }>("/api/latest");
  return data.filenames;
}
