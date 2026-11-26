import { Interactive } from "remotion";

export const Logo = ({ size }: { readonly size: number }) => (
  <Interactive.Svg name="Logo" width={size} height={size} viewBox="0 0 24 24">
    <rect width="24" height="24" rx="6" fill="#F2C94C" />
    <Interactive.Path
      name="Logo h"
      fill="#000000"
      d="M7 5h2.5v5.6c.8-1 2-1.6 3.5-1.6 2.6 0 4 1.6 4 4.4V19h-2.5v-5.3c0-1.6-.7-2.4-2-2.4-1.6 0-3 1.1-3 3.1V19H7z"
    />
  </Interactive.Svg>
);
