import type { CSSProperties, ReactNode } from "react";

type Props = {
  readonly width: number;
  readonly height: number;
  readonly chrome: string;
  readonly background: string;
  readonly url?: string;
  readonly title?: string;
  readonly style?: CSSProperties;
  readonly children: ReactNode;
};

export const AppWindow = ({
  width,
  height,
  chrome,
  background,
  url,
  title,
  style,
  children,
}: Props) => (
  <div
    style={{
      width,
      height,
      display: "flex",
      flexDirection: "column",
      borderRadius: 12,
      overflow: "hidden",
      backgroundColor: background,
      border: "1px solid rgba(255, 255, 255, 0.12)",
      boxShadow: "0 24px 48px rgba(0, 0, 0, 0.6)",
      fontFamily: "Inter",
      ...style,
    }}
  >
    <div
      style={{
        height: 44,
        flexShrink: 0,
        display: "flex",
        alignItems: "center",
        gap: 8,
        padding: "0 16px",
        backgroundColor: chrome,
        borderBottom: "1px solid rgba(255, 255, 255, 0.08)",
      }}
    >
      {["#FF5F57", "#FEBC2E", "#28C840"].map((color) => (
        <span
          key={color}
          style={{
            width: 12,
            height: 12,
            borderRadius: 9999,
            backgroundColor: color,
          }}
        />
      ))}
      {url ? (
        <span
          style={{
            flex: 1,
            margin: "0 24px",
            padding: "4px 0",
            borderRadius: 6,
            backgroundColor: "rgba(255, 255, 255, 0.08)",
            color: "#9DA3AE",
            fontSize: 15,
            textAlign: "center",
          }}
        >
          {url}
        </span>
      ) : (
        <span
          style={{
            flex: 1,
            marginRight: 52,
            color: "#9DA3AE",
            fontSize: 15,
            fontWeight: 600,
            textAlign: "center",
          }}
        >
          {title}
        </span>
      )}
    </div>
    <div style={{ flex: 1, display: "flex", overflow: "hidden" }}>
      {children}
    </div>
  </div>
);
