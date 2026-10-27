import { TagBubble } from "./App";
import { Card, CardHeader, CardTitle } from "./components/ui/card";
import { Skeleton } from "./components/ui/skeleton";

export function ImageCard({
  imageName,
  title,
  tags,
}: {
  imageName: string;
  title: string;
  tags: string[];
}) {
  const imageUrl = `/images/${imageName}`;

  return (
    <Card>
      <img src={imageUrl} />
      <CardHeader className="flex flex-row gap-1">
        <CardTitle>{title}</CardTitle>
        <div className="flex flex-row gap-1">
          {tags.map((t) => (
            <TagBubble title={t} />
          ))}
        </div>
      </CardHeader>
    </Card>
  );
}

export function SkeletonCard() {
  return (
    <Card>
      <Skeleton className="h-64 w-full aspect-video" />
      <CardHeader>
        <div className="flex flex-row gap-1">
          <Skeleton className="h-4 w-32" />
          <div className="flex flex-row gap-1 ">
            <Skeleton className="h-4 w-16" />
            <Skeleton className="h-4 w-16" />
          </div>
        </div>
      </CardHeader>
    </Card>
  );
}

export function SkeletonCardList({ n }: { n: number }) {
  return (
    <ul className="flex flex-col gap-3">
      {Array.from({ length: n }, (_, i) => (
        <li key={i}>
          <SkeletonCard />
        </li>
      ))}
    </ul>
  );
}
