<script lang="ts">
    import {apiConfig, setBasicAuth} from "./api/client";
    import {Configuration, SpaceControllerApi} from "./api/gen";

    let count: number = $state(0)
    const increment = () => {
        count += 1
    }

    async function way2() {
        setBasicAuth("admin", "test123");

        const api = new SpaceControllerApi(apiConfig());

        const response = await api.getSpaces();
        console.log(response);
    }

    async function way1() {
        const api = new SpaceControllerApi(
            new Configuration({
                //basePath: import.meta.env.VITE_API_URL
                basePath: "http://localhost:8080"
            })
        );

        const response = await api.getSpaces({
            headers: {
                Authorization: `Basic ${btoa('admin:test123')}`
            }
        });
        console.log(response);
    }

    way2();
</script>

<button onclick={increment}>
    count is {count}
</button>
