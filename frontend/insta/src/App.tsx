import {
  QueryClient,
  QueryClientProvider,
  useQuery,
} from "@tanstack/react-query";
import "./App.css";
import axios from "axios";
import { Badge } from "./components/ui/badge";
import { Button } from "./components/ui/button";
import { ImageCard, SkeletonCardList } from "./Card";
import {
  Dialog,
  DialogContent,
  DialogTrigger,
} from "./components/ui/dialog";

import { UploadForm } from "./UploadForm";

export const PlusIcon = (props: React.SVGProps<SVGSVGElement>) => (
  <svg
    xmlns="http://www.w3.org/2000/svg"
    width="21"
    height="24"
    viewBox="0 0 448 512"
    {...props}
  >
    <path
      fill="currentColor"
      d="M256 80c0-17.7-14.3-32-32-32s-32 14.3-32 32v144H48c-17.7 0-32 14.3-32 32s14.3 32 32 32h144v144c0 17.7 14.3 32 32 32s32-14.3 32-32V288h144c17.7 0 32-14.3 32-32s-14.3-32-32-32H256z"
    />
  </svg>
);

export const TagsIcon = (props: React.SVGProps<SVGSVGElement>) => (
  <svg
    xmlns="http://www.w3.org/2000/svg"
    width="24"
    height="24"
    viewBox="0 0 512 512"
    {...props}
  >
    <path
      fill="currentColor"
      d="m345 39.1l127.8 129.3c52.4 53 52.4 138.2 0 191.2l-112 113.3c-9.3 9.4-24.5 9.5-33.9.2s-9.5-24.5-.2-33.9l111.9-113.3c33.9-34.3 33.9-89.4 0-123.7L310.9 72.9c-9.3-9.4-9.2-24.6.2-33.9s24.6-9.2 33.9.2zM0 229.5V80c0-26.5 21.5-48 48-48h149.5c17 0 33.3 6.7 45.3 18.7l168 168c25 25 25 65.5 0 90.5L277.3 442.7c-25 25-65.5 25-90.5 0l-168-168C6.7 262.7 0 246.5 0 229.5M144 144a32 32 0 1 0-64 0a32 32 0 1 0 64 0"
    />
  </svg>
);

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
      <div className="m-4">
        <div className="fixed top-0 mt-4">
          <div className="flex flex-col">
            <UploadButton />
            <Button size="icon-lg">
              <TagsIcon />
            </Button>
          </div>
        </div>
        <div className="m-5 mx-auto w-full max-w-xl">
          <Feed />
        </div>
      </div>
    </QueryClientProvider>
  );
}

export default App;

export function TagBubble({ title }: { title: string }) {
  return <Badge variant="outline">#{title}</Badge>;
}

export function TopBar() {
  return <div className="w-full h-10 border-b-2"></div>;
}

export function Feed() {
  const { isPending, isError, data, error } = useQuery({
    queryKey: ["feed"],
    queryFn: getLatest,
  });

  if (isPending) return <SkeletonCardList n={20} />;
  if (isError) return <span>Error: {error.message}</span>;

  return (
    <ul className="flex flex-col gap-3">
      {data.map((e) => (
        <li key={e.id}>
          <ImageCard title={e.title} tags={e.tags} imageId={e.id} />
        </li>
      ))}
    </ul>
  );
}

interface LatestResponseElement {
  id: string;
  title: string;
  tags: string[];
  filesize: number;
  mimetype: string;
  size_x: number;
  size_y: number;
  created_at: string; //TODO use correct type.
}

async function getLatest(): Promise<LatestResponseElement[]> {
  //timeout to show skelleton. Leaving zombie code in for demo purposes
  // await new Promise((f) => setTimeout(f, 2000));
  const { data } = await axios.get<{ metadata: LatestResponseElement[] }>(
    "/api/latest",
  );
  return data.metadata;
}

export function UploadButton() {
  return (
    <Dialog>
      <DialogTrigger
        render={
          <Button size="icon-lg">
            <PlusIcon />
          </Button>
        }
      />
      <DialogContent className="ms:max-w-sm">
        <UploadForm />
      </DialogContent>
    </Dialog>
  );
}
