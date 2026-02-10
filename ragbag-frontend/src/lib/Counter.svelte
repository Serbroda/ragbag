<script lang="ts">
    import {apiConfig, login} from "./api/client";
    import {SpaceApi} from "ragbag-frontend-sdk";

    let count: number = $state(0)
    const increment = () => {
        count += 1
    }

    async function way3() {
        await login("admin", "test123");

        const api = new SpaceApi(apiConfig());
        const response = await api.getSpaces();
        console.log(response);
    }

    way3();
</script>

<button onclick={increment}>
    count is {count}
</button>
