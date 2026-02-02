package de.serbroda.ragbag.service;

import de.serbroda.ragbag.model.Collection;
import de.serbroda.ragbag.repository.CollectionRepository;
import lombok.RequiredArgsConstructor;
import org.springframework.stereotype.Service;

@RequiredArgsConstructor
@Service
public class CollectionService {

    private final CollectionRepository collectionRepository;

    public Collection getCollection(String id) {
        return collectionRepository.findById(id).orElse(null);
    }

    public Collection createCollection(Collection collection) {
        return collectionRepository.save(collection);
    }
}
