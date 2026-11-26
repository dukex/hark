import "./fonts";
import { Composition } from "remotion";
import { PromoLinkedIn } from "./linkedin/PromoLinkedIn";

export const RemotionRoot = () => (
  <Composition
    id="PromoLinkedIn"
    component={PromoLinkedIn}
    durationInFrames={900}
    fps={30}
    width={1080}
    height={1350}
  />
);
