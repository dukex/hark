import "./fonts";
import { Composition } from "remotion";
import { PromoLinkedIn } from "./linkedin/PromoLinkedIn";
import { Promo } from "./Promo";

export const RemotionRoot = () => (
  <>
    <Composition
      id="Promo"
      component={Promo}
      durationInFrames={1800}
      fps={30}
      width={1920}
      height={1080}
    />
    <Composition
      id="PromoLinkedIn"
      component={PromoLinkedIn}
      durationInFrames={900}
      fps={30}
      width={1080}
      height={1350}
    />
  </>
);
