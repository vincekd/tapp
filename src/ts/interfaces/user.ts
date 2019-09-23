import { Media } from "./media";

export interface User {
  ScreenName: string;
  Id: number;
  Url: string;
  ProfileImageUrlHttps: string;
  Name: string;
  Description: string;
  Followers: number;
  Following: number;
  TweetCount: number;
  Location: string;
  Verified: boolean;
  Link: string;
  Updated: number;
  Media: Media;
}
