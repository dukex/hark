import type { CSSProperties, ReactNode } from "react";
import { AppWindow } from "./AppWindow";

export const ClaudeDesktop = ({
  width,
  height,
  style,
  children,
}: {
  readonly width: number;
  readonly height: number;
  readonly style?: CSSProperties;
  readonly children: ReactNode;
}) => (
  <AppWindow
    width={width}
    height={height}
    style={style}
    title="Claude"
    chrome="#1F1E1D"
    background="#262624"
  >
    <div
      style={{
        flex: 1,
        display: "flex",
        flexDirection: "column",
        padding: "28px 36px",
        color: "#F5F4EF",
      }}
    >
      <div style={{ flex: 1, display: "flex", flexDirection: "column", gap: 20 }}>
        {children}
      </div>
      <div
        style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "space-between",
          padding: "20px 24px",
          borderRadius: 18,
          backgroundColor: "#30302E",
          border: "1px solid #3E3E3A",
          color: "#8F8D86",
          fontSize: 22,
        }}
      >
        Reply to Claude…
        <span
          style={{
            width: 40,
            height: 40,
            borderRadius: 10,
            backgroundColor: "#D97757",
            color: "#FFFFFF",
            fontSize: 24,
            fontWeight: 700,
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
          }}
        >
          ↑
        </span>
      </div>
    </div>
  </AppWindow>
);

export const UserMessage = ({ children }: { readonly children: ReactNode }) => (
  <span
    style={{
      alignSelf: "flex-end",
      maxWidth: "80%",
      padding: "14px 20px",
      borderRadius: 14,
      backgroundColor: "#141413",
      fontSize: 28,
      lineHeight: 1.35,
    }}
  >
    {children}
  </span>
);

export const ToolLine = ({
  name,
  target,
  stat,
}: {
  readonly name: string;
  readonly target: string;
  readonly stat: string;
}) => (
  <span
    style={{
      display: "flex",
      alignItems: "center",
      gap: 12,
      fontFamily: "JetBrains Mono",
      fontSize: 26,
      color: "#C2C0B6",
    }}
  >
    <span
      style={{
        width: 10,
        height: 10,
        borderRadius: 9999,
        backgroundColor: "#7FB069",
      }}
    />
    <b style={{ color: "#F5F4EF" }}>{name}</b>({target})
    <span style={{ color: "#7FB069" }}>{stat}</span>
  </span>
);

export const MonitorEvent = ({
  command,
  line,
  style,
}: {
  readonly command: string;
  readonly line: string;
  readonly style?: CSSProperties;
}) => (
  <span
    style={{
      display: "flex",
      flexDirection: "column",
      gap: 6,
      padding: "12px 18px",
      borderRadius: 10,
      borderLeft: "4px solid #D97757",
      backgroundColor: "#1F1E1D",
      fontFamily: "JetBrains Mono",
      ...style,
    }}
  >
    <span style={{ fontSize: 22, color: "#8F8D86" }}>Monitor · {command}</span>
    <span style={{ fontSize: 28, color: "#F5F4EF" }}>{line}</span>
  </span>
);
