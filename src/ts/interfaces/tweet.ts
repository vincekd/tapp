import { Media } from "./media";

export interface Tweet {
    Ratio: number;
    IdStr: string;
    Faves: number;
    Rts: number;
    Id: number;
    Created: number;
    Updated: number;
    Text: string;
    Url: string;
    Media: Media[];
}
