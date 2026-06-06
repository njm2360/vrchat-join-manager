import { Link } from "@mui/material";
import { usePlayerDetailDialog } from "@/components/usePlayerDetailDialog";

interface Props {
  userId: string;
  displayName: string;
  instanceId?: number;
  // 行クリックを兼ねるテーブル内で使う場合に伝播を止める
  stopPropagation?: boolean;
}

export default function PlayerLink({ userId, displayName, instanceId, stopPropagation }: Props) {
  const { open } = usePlayerDetailDialog();
  return (
    <Link
      component="button"
      underline="hover"
      onClick={(e) => {
        if (stopPropagation) e.stopPropagation();
        open({ userId, displayName, instanceId });
      }}
    >
      {displayName}
    </Link>
  );
}
