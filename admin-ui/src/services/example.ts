import qs from "qs";
import { Http } from "@/utilities/http";

export class ExampleService {
	async startPageView(payload: {
		example: string;
	}): Promise<{ id: string; created: boolean }> {
		return Http.post<{ id: string; created: boolean }>(
			"/example/path/",
			payload,
		);
	}
}
