package de.serbroda.ragbag.mapper;

import de.serbroda.ragbag.generated.model.CollectionDto;
import de.serbroda.ragbag.service.CollectionService;
import org.springframework.stereotype.Component;

@Component
public class CollectionDtoMapper {

    public CollectionDto toDto(CollectionService.CollectionNode node) {
        return new CollectionDto.Builder()
                .id(node.id())
                .name(node.name())
                .parentId(node.parentId())
                .children(node.children().stream().map(this::toDto).toList())
                .build();
    }
}
