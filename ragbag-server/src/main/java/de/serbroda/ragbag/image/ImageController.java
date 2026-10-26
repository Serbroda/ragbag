package de.serbroda.ragbag.image;

import static de.serbroda.ragbag.shared.ApiConstants.PUBLIC_API_PREFIX;

import de.serbroda.ragbag.generated.api.ImageApi;
import de.serbroda.ragbag.shared.exception.ResourceNotFoundException;
import java.time.Duration;
import lombok.RequiredArgsConstructor;
import org.springframework.core.io.ByteArrayResource;
import org.springframework.core.io.Resource;
import org.springframework.http.CacheControl;
import org.springframework.http.MediaType;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RequiredArgsConstructor
@RestController
@RequestMapping(PUBLIC_API_PREFIX)
public class ImageController implements ImageApi {

    private final ImageService imageService;

    // Public, so <img> tags work without a bearer token; ids are random and change on every upload
    @Override
    public ResponseEntity<Resource> getImage(String imageId) {
        Image image = imageService
                .findImage(imageId)
                .orElseThrow(() -> new ResourceNotFoundException("Image not found: " + imageId));
        return ResponseEntity.ok()
                .contentType(MediaType.parseMediaType(image.getContentType()))
                .cacheControl(
                        CacheControl.maxAge(Duration.ofDays(365)).cachePublic().immutable())
                // Opened directly, the image must never be able to run anything on our origin
                .header("Content-Security-Policy", "default-src 'none'; sandbox")
                .body(new ByteArrayResource(image.getData()));
    }
}
