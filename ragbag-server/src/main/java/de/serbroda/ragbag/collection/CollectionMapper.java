package de.serbroda.ragbag.collection;

import de.serbroda.ragbag.generated.model.CollectionDto;
import org.springframework.stereotype.Component;

@Component
public class CollectionMapper {

    public CollectionDto toDto(CollectionService.CollectionNode node) {
        return new CollectionDto.Builder()
                .id(node.id())
                .name(node.name())
                .parentId(node.parentId())
                .children(node.children().stream().map(this::toDto).toList())
                .build();
    }
}
